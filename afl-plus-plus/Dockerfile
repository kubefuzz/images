FROM alpine:3.11

ARG GIT_URL="https://github.com/AFLplusplus/AFLplusplus.git"
ARG GIT_TAG="2.65c"

ARG CC="gcc"
ARG CXX="g++"

# Should match the installed LLVM version (v9), but only v10 has this necessary fix:
# https://github.com/llvm/llvm-project/commit/7e8541f3df064e133c9f99e889682833b17d5f8e
ARG LLVM_AFL_DRIVER="https://raw.githubusercontent.com/llvm/llvm-project/llvmorg-10.0.0/compiler-rt/lib/fuzzer/afl/afl_driver.cpp"

# It is not possible to set `echo core > /proc/sys/kernel/core_pattern` inside an unprivileged container,
# since `/proc` is a shared host directory. The setting would therefore change the host configuration.
# https://github.com/moby/moby/issues/19289
# This option has to be activated if AFL is run in an environment were the `core_pattern` cannot be changed.
# https://github.com/AFLplusplus/AFLplusplus/blob/2.65c/src/afl-fuzz-init.c#L1539-L1552
# In minikube this is not an issue, since the default VM has a valid `core_pattern` setting.
# ENV AFL_I_DONT_CARE_ABOUT_MISSING_CRASHES 1

# If `/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor` is not set to `performance`, AFL refuses to start.
# https://github.com/google/AFL/blob/v2.56b/docs/env_variables.txt#L92
# https://github.com/AFLplusplus/AFLplusplus/blob/2.65c/src/afl-fuzz-init.c#L1681-L1699
# This is mostly a problem on desktop and laptop systems (where power-saving is important).
# In minikube this is not an issue, since the default VM has no `scaling_governor` configured.
# ENV AFL_SKIP_CPUFREQ 1

# In a default Kubernetes environment, each AFL process is isolated in a pod container. This seems to cause an
# issue with AFL, since it will always select CPU#0 instead of using anohter free core. Even when a container
# has a resource limit of 1 CPU, all node CPUs are passed into the container and scheduled via CFS.
# To avoid this issue, we tell AFL to not bind itself to a specific CPU.
# https://github.com/AFLplusplus/AFLplusplus/blame/2.65c/docs/env_variables.md#L271-L273
# Sidenote: this is not needed in a native Docker environment when assigning a specific CPU with `--cpuset-cpus`
#           or if the Kubernetes kubelet is configured with `cpuManagerPolicy: static`.
# ENV AFL_NO_AFFINITY 1

# Disable x86 support in order to avoid error messages.
ENV AFL_NO_X86 1

# Disable AFL++ Python support to keep a small images size.
ENV NO_PYTHON 1

# Directory to store build artifacts (fuzz targets, dictionaries, options files, seed corpus archives)
ENV OUT /fuzz/out

# Directory to checkout source files
ENV SRC /fuzz/src

# Directory for input files
ENV INPUT /fuzz/input

# File name for fuzz-target binary
ENV TARGET $OUT/target

# Library for libFuzzer targets
ENV FUZZER_LIB /fuzz/libAFLDriver.a

# Create folders
RUN mkdir -p $OUT $SRC $INPUT

# Install runtime dependencies
RUN apk add --no-cache \
    # Console display library
    ncurses \
    # Required to copy the sync directory with Kubernetes API
    tar

# Build AFL
RUN echo "Building AFL++ ..."; \
    # Install build dependencies:
    # - build-base, clang-dev, llvm9-dev:
    #   Required for building AFL++.
    # - ca-certificates, git:
    #   Required for downloading source code.
    # - coreutils:
    #   Required for `llvm_mode` compilation.
    apk add --no-cache --virtual .build-deps \
            build-base \
            ca-certificates \
            clang-dev \
            coreutils \
            git \
            llvm9-dev; \
    \
    # Link clang in LLVM bindir to satisfy `llvm_mode` makefile:
    # https://github.com/AFLplusplus/AFLplusplus/blob/2.65c/llvm_mode/GNUmakefile#L82-L87
    ln -s $(which clang) $(llvm-config --bindir); \
    ln -s $(which clang++) $(llvm-config --bindir); \
    \
    # Checkout AFL++
    mkdir -p /usr/src; \
    cd /usr/src; \
    \
    git clone --branch ${GIT_TAG} ${GIT_URL}; \
    \
    # Build AFL++ with llvm_mode (make all: just the main binaries)
    cd AFLplusplus; \
    make all -j "$(nproc)"; \
    cd ./llvm_mode && make -j "$(nproc)" && cd ./../; \
    make install; \
    \
    # Build LLVM libFuzzer AFL driver
    # https://github.com/google/fuzzbench/blob/95e39a6e6a41cb8476b307594dc1d3e785c0e3eb/fuzzers/aflplusplus/builder.Dockerfile
    echo "Building LLVM libFuzzer AFL driver ..."; \
    wget ${LLVM_AFL_DRIVER} -O ./afl_driver.cpp; \
    clang++ -v -std=c++11 -O2 -c ./afl_driver.cpp; \
    ar rUu ${FUZZER_LIB} afl_driver.o; \
    cd ./../; \
    \
    # Cleanup
    echo "Cleaning up ..."; \
    rm -rf ./AFLplusplus; \
    apk del --no-network .build-deps

WORKDIR /fuzz

# https://github.com/AFLplusplus/AFLplusplus/blob/2.65c/docs/parallel_fuzzing.md
# Each fuzzer will keep its state in a separate subdirectory, like so:
# /fuzz/sync/fuzzer01/
# Each instance will periodically rescan the top-level sync directory for
# any test cases found by other fuzzers - and will incorporate them into
# its own fuzzing when they are deemed interesting enough.
VOLUME /fuzz/sync

COPY entrypoint.sh healthcheck.sh /usr/local/bin/

# https://docs.docker.com/engine/reference/builder/#healthcheck
# https://github.com/docker-library/faq#healthcheck
# Important: Kubernetes doesn't support this feature because it would
#            conflict with its own Liveness and Readiness features:
#            https://github.com/kubernetes/kubernetes/pull/50796
HEALTHCHECK --interval=1m --timeout=5s CMD healthcheck.sh

ENTRYPOINT ["entrypoint.sh"]

# https://github.com/AFLplusplus/AFLplusplus/blob/2.65c/docs/QuickStartGuide.md
# Default command
# If the program reads from stdin, run 'afl-fuzz' like so:
#
#   ./afl-fuzz -i testcase_dir -o findings_dir -- \
#     /path/to/tested/program [...program's cmdline...]
#
# If the program takes input from a file, you can put @@ in the program's
# command line; AFL will put an auto-generated file name in there for you.
CMD ["afl-fuzz", "-i /fuzz/input", "-o /fuzz/sync"]