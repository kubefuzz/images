FROM kubefuzz/afl-plus-plus

# This image is based on the woff2-benchmark from google/fuzzbench:
# https://github.com/google/fuzzbench/tree/9f6352d899c7e832775c8e1e01bff1186e8af9da/benchmarks/woff2-2016-05-06

# How to run:
# afl-fuzz -i ./input/ -o ./sync/ -- ./out/target 2147483647

ARG GIT_URL="https://github.com/google/woff2.git"
ARG GIT_TAG="v1.0.2"

ENV CC  afl-clang-fast
ENV CXX afl-clang-fast++

# Install runtime dependencies
RUN apk add --no-cache \
    libstdc++ \
    libgcc

# Build libjpeg-turbo
RUN echo "Preparing woff2 (${GIT_TAG}) target ..."; \
    # Install build dependencies
    apk add --virtual .build-deps \
            autoconf \
            automake \
            build-base \
            clang \
            git \
            libtool \
            pkgconfig \
            wget; \
    \
    # Download
    git clone --recurse-submodules --branch ${GIT_TAG} ${GIT_URL} ${SRC}; \
    \
    # Build lib
    cd ${SRC}; \
    make clean all -j "$(nproc)"; \
    \
    # Build target
    $CXX -v $FUZZER_LIB ./src/convert_woff2ttf_fuzzer.a -o $TARGET; \
    \
    # Download and copy seeds
    git clone https://github.com/google/oss-fuzz.git && cd ./oss-fuzz; \
    git checkout 7643d953146b4d1840e4581b00041ecf432250d7; \
    cp ./projects/woff2/corpus/* $INPUT; \
    cd ./../../; \
    \
    # Cleanup
    rm -rf ${SRC}; \
    apk del --no-network .build-deps