FROM kubefuzz/afl-plus-plus

# This image is based on the libjpeg-trubo-benchmark from google/fuzzbench:
# https://github.com/google/fuzzbench/tree/4dfc4b0b27a28208a0cf4b4623e8b84a8278dbba/benchmarks/libjpeg-turbo-07-2017

# How to run:
# afl-fuzz -i ./input/ -o ./sync/ -- ./out/target 2147483647

ARG GIT_URL="https://github.com/libjpeg-turbo/libjpeg-turbo.git"
ARG GIT_TAG="1.5.2"

ENV CC  afl-clang-fast
ENV CXX afl-clang-fast++

# Install runtime dependencies
RUN apk add --no-cache \
    libstdc++ \
    libgcc

# Build libjpeg-turbo
RUN echo "Preparing libjpeg-trubo (${GIT_TAG}) target ..."; \
    # Install build dependencies
    apk add --virtual .build-deps \
            autoconf \
            automake \
            build-base \
            clang \
            git \
            libtool \
            nasm \
            wget; \
    \
    # Download
    git clone --branch ${GIT_TAG} ${GIT_URL} ${SRC}; \
    \
    # Build lib
    cd ${SRC}; \
    autoreconf -fiv; \
    ./configure; \
    make -j "$(nproc)"; \
    \
    # Build target
    wget https://raw.githubusercontent.com/google/fuzzbench/218882fa25c3e8cae45e14aa5b19954c4627b387/benchmarks/libjpeg-turbo-07-2017/libjpeg_turbo_fuzzer.cc -O target.cc; \
    $CXX -v -std=c++11 target.cc -I . ./.libs/libturbojpeg.a $FUZZER_LIB -o $TARGET; \
    cd ./../; \
    \
    # Download and copy seeds
    wget https://raw.githubusercontent.com/google/fuzzbench/a39df8809ea6aa330a7a3124f1b1d3d17fee7041/benchmarks/libjpeg-turbo-07-2017/seeds/seed.jpg -P $INPUT; \
    \
    # Cleanup
    rm -rf ${SRC}; \
    apk del --no-network .build-deps