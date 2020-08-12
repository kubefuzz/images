package main

import (
	"bytes"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"strings"
	"time"
)

const AflSyncDirectory string = "sync/"
const K8sDefaultNamespace string = "kubefuzz"
const K8sLabelSelector string = "app=kubefuzz"
const K8sMasterPodPrefix string = "afl-master"
const K8sWorkerPodPrefix string = "afl-worker"
const WaitUntilValidate time.Duration = 5 // Value in seconds

type K8sClient struct {
	ClientSet *kubernetes.Clientset
	Config    *rest.Config
}

func main() {
	client := getK8sClient()

	K8sNamespace := getEnv("POD_NAMESPACE", K8sDefaultNamespace)
	syncStatsOnly := os.Getenv("SYNC_STATS_ONLY") == "1"

	// Get all pods with label `app: kubefuzz` in `kubefuzz` namespace
	pods, err := client.ClientSet.CoreV1().Pods(K8sNamespace).List(metav1.ListOptions{LabelSelector: K8sLabelSelector})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("There are %d pods in the %s namespace:\n", len(pods.Items), K8sNamespace)
	for _, pod := range pods.Items {
		fmt.Printf("- %s \n", pod.Name)
	}

	fmt.Println()
	fmt.Printf("Starting sync ...\n")

	if syncStatsOnly {
		// Sync `fuzzer_stats` file
		// Needed to collect metrics from all fuzzers on the master instance
		fmt.Printf("Syncing only 'fuzzer_stats'! (SYNC_ONLY_STATS)\n")
		fmt.Println()
		syncFuzzerStats(client, pods)
	} else {
		// Sync fuzzing directories
		// https://github.com/google/AFL/blob/v2.56b/docs/parallel_fuzzing.txt#L89
		fmt.Println()
		sync(client, pods)
		whatsUp(client, pods)
	}

	fmt.Println()
	fmt.Printf("Sync finished!")
}

func sync(client *K8sClient, pods *corev1.PodList) {
	for _, sourcePod := range pods.Items {
		fmt.Printf("Copying sync directory from pod %s \n", sourcePod.Name)

		// We have to copy the sync folder before we can archive it, because otherwise `tar` will complain about
		// changing files and return exit code 1, which results in a panic.
		// After copying the folder, we output the archive to stdout and write it to a buffer.
		cmdArchive := fmt.Sprintf("cp -r -T %[1]s%[2]s/ %[2]s/ && tar -czf %[2]s.tar.gz %[2]s/ && cat %[2]s.tar.gz", AflSyncDirectory, sourcePod.Name)
		syncDirArchive, _ := exec(*client, sourcePod, []string{"sh", "-c", cmdArchive}, nil)

		for _, targetPod := range pods.Items {
			if sourcePod.Name == targetPod.Name {
				continue
			}

			fmt.Printf("- Writing sync directory to pod %s \n", targetPod.Name)

			// Here we use the buffer from above and pass it to the pod via stdin.
			// We don't overwrite old files (--skip-old-files) because according to the docs, this is not advisable:
			// https://github.com/google/AFL/blob/v2.56b/docs/parallel_fuzzing.txt#L137
			reader := bytes.NewReader(syncDirArchive.Bytes())
			_, stderr := exec(*client, targetPod, []string{"tar", "--skip-old-files", "-xzf", "/dev/stdin", "--directory", AflSyncDirectory}, reader)
			if stderr != "" {
				fmt.Printf("\nError:\n%s\n", stderr)
			}
		}

		// Delete the copied directory and the created archive after we synced it.
		cmdCleanup := fmt.Sprintf("rm -rf %[1]s/ && rm %[1]s.tar.gz", sourcePod.Name)
		stdout, stderr := exec(*client, sourcePod, []string{"sh", "-c", cmdCleanup}, nil)
		fmt.Printf("%s\n", stdout.String())
		if stderr != "" {
			fmt.Printf("\nError:\n%s\n", stderr)
		}
	}
}

func syncFuzzerStats(client *K8sClient, pods *corev1.PodList) {
	for _, sourcePod := range pods.Items {
		fmt.Printf("Copying 'fuzzer_stats' file from pod %s \n", sourcePod.Name)

		sourcePath := fmt.Sprintf("%[1]s%[2]s", AflSyncDirectory, sourcePod.Name)
		fuzzerStatsPath := fmt.Sprintf("%s/fuzzer_stats", sourcePath)
		cmdFuzzerStats := fmt.Sprintf("cat %s", fuzzerStatsPath)
		fuzzerStatsContent, _ := exec(*client, sourcePod, []string{"sh", "-c", cmdFuzzerStats}, nil)

		for _, targetPod := range pods.Items {
			if sourcePod.Name == targetPod.Name {
				continue
			}

			// Don't sync stats with other workers.
			// Stats are only needed on the master instance (where metrics are collected).
			// The stats sync should also be fast, since metrics are collected every minute.
			if strings.HasPrefix(targetPod.Name, K8sWorkerPodPrefix) {
				continue
			}

			fmt.Printf("- Writing 'fuzzer_stats' file to pod %s \n", targetPod.Name)

			// Make sure the destination directory exists
			exec(*client, targetPod, []string{"mkdir", "--parents", sourcePath}, strings.NewReader(""))

			reader := bytes.NewReader(fuzzerStatsContent.Bytes())
			_, stderr := exec(*client, targetPod, []string{"cp", "/dev/stdin", fuzzerStatsPath}, reader)
			if stderr != "" {
				fmt.Printf("\nError:\n%s\n", stderr)
			}
		}
	}
}

// Output a status report from all fuzzing instances.
// This way we will know if we found a crash on one of the pods.
// Crashing inputs are *not* automatically propagated to the master instance:
// https://github.com/google/AFL/blob/v2.56b/docs/parallel_fuzzing.txt#L184
func whatsUp(client *K8sClient, pods *corev1.PodList) {
	fmt.Printf("Validating sync in %d seconds ...\n", WaitUntilValidate)
	time.Sleep(WaitUntilValidate * time.Second)
	fmt.Printf("Getting status report ...\n\n")
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, K8sMasterPodPrefix) {
			fmt.Printf("Stats on %s:\n\n", pod.Name)
			command := fmt.Sprintf("afl-whatsup -s %s", AflSyncDirectory)
			stdout, stderr := exec(*client, pod, []string{"sh", "-c", command}, nil)
			fmt.Printf("%s\n", stdout.String())
			if stderr != "" {
				fmt.Printf("\nError:\n%s\n", stderr)
			}
		}
	}
}

// Get environment variable.
// Returns fallback if environment variable is not found.
func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Get Kubernetes ClientSet and Config
// Inspired by https://github.com/nuvo/skbn/blob/master/pkg/skbn/kube.go
func getK8sClient() *K8sClient {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// Create the ClientSet
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return &K8sClient{ClientSet: clientSet, Config: config}
}

// Execute a command in a pod
// Inspired by https://github.com/nuvo/skbn/blob/master/pkg/skbn/kube.go
func exec(client K8sClient, pod corev1.Pod, command []string, stdin io.Reader) (bytes.Buffer, string) {
	clientSet, config := client.ClientSet, client.Config

	// Get the name of the first container in the pod. This name is required for pods which have multiple containers.
	// This is the case for the AFL master pod, which has a sidecar container for collecting metrics.
	podContainerName := pod.Spec.Containers[0].Name

	// Create API POST request
	req := clientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		Param("container", podContainerName).
		SubResource("exec")

	// Add scheme
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(err.Error())
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Command: command,
		Stdin:   stdin != nil,
		Stdout:  true,
		Stderr:  true,
		TTY:     false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		panic(err.Error())
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})

	if err != nil {
		fmt.Printf("exec stderr: %s\n", stderr.String())
		panic(err.Error())
	}

	return stdout, stderr.String()
}
