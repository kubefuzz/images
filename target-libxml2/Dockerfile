FROM kubefuzz/afl-plus-plus

# This image is based on the libxml2-benchmark from google/fuzzbench:
# https://github.com/google/fuzzbench/tree/4dfc4b0b27a28208a0cf4b4623e8b84a8278dbba/benchmarks/libxml2-v2.9.2

# How to run:
# afl-fuzz -i ./input/ -o ./sync/ -x ./out/target.dict -- ./out/target 2147483647

ARG GIT_URL="https://github.com/GNOME/libxml2.git"
ARG GIT_TAG="v2.9.2"

ENV CC  afl-clang-fast
ENV CXX afl-clang-fast++

# Install runtime dependencies
RUN apk add --no-cache \
    libstdc++ \
    libgcc

# Build libxml2
RUN echo "Preparing libxml2 (${GIT_TAG}) target ..."; \
    # Install build dependencies
    apk add --virtual .build-deps \
            autoconf \
            automake \
            build-base \
            clang-dev \
            git \
            libtool \
            pkgconfig \
            wget; \
    \
    # Download
    git clone --branch ${GIT_TAG} ${GIT_URL} ${SRC}; \
    \
    # Build lib
    cd ${SRC}; \
    ./autogen.sh; \
    ./configure --with-http=no --without-python --with-threads=no --with-zlib=no --with-lzma=no; \
    make -j "$(nproc)"; \
    \
    # Build target
    wget https://raw.githubusercontent.com/google/fuzzbench/218882fa25c3e8cae45e14aa5b19954c4627b387/benchmarks/libxml2-v2.9.2/target.cc; \
    $CXX -v -std=c++11 target.cc -I ./include ./.libs/libxml2.a $FUZZER_LIB -o $TARGET; \
    cd ./../; \
    \
    # Download and copy dictionary
    # https://github.com/AFLplusplus/AFLplusplus/blob/2.65c/dictionaries/README.md
    wget https://raw.githubusercontent.com/AFLplusplus/AFLplusplus/2.65c/dictionaries/xml.dict -O ${TARGET}.dict; \
    \
    # AFL needs at least one non-empty seed to start
    echo "default" > ${INPUT}/seed; \
    \
    # Cleanup
    rm -rf ${SRC}; \
    apk del --no-network .build-deps