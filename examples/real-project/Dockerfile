FROM debian:12.7 as builder

ARG CONAN_ARTIFACTORY="172.18.1.111:8081"
ARG VERSION

ENV DEBIAN_FRONTEND=noninteractive \
    CONAN_PATH=.conan2 \
    ORACLE_HOME=/usr/lib/oracle/12.2/client64

RUN apt-get update && \
    apt-get -y install \
    git gcc-12 g++-12 make cmake ninja-build \
    autoconf automake libtool binutils \
    libstdc++-12-dev \
    libdigest-sha-perl libipc-run-perl \
    google-perftools glibc-source libgoogle-perftools-dev \
    cppcheck libev-dev libpcre3-dev \
    gettext flex \
    python3 python3-pip

RUN git clone -v https://github.com/conan-io/conan.git conan-io\
    && cd conan-io \
    && pip3 install --break-system-packages -e . \
    && cd ..

RUN apt -y install libaio1 alien wget

RUN mkdir build && \
    cd build

RUN wget http://rpm.svc/repository/3rd_party/8/x86_64/Packages/oracle-instantclient12.2-basic-12.2.0.1.0-1.x86_64.rpm && \
    wget http://rpm.svc/repository/3rd_party/8/x86_64/Packages/oracle-instantclient12.2-devel-12.2.0.1.0-1.x86_64.rpm && \
    wget http://rpm.svc/repository/3rd_party/8/x86_64/Packages/oracle-instantclient12.2-sqlplus-12.2.0.1.0-1.x86_64.rpm && \
    wget http://rpm.svc/repository/3rd_party/8/x86_64/Packages/oracle-instantclient12.2-tools-12.2.0.1.0-1.x86_64.rpm && \
    alien -i oracle-instantclient12.2-basic-12.2.0.1.0-1.x86_64.rpm &&\
    alien -i oracle-instantclient12.2-devel-12.2.0.1.0-1.x86_64.rpm &&\
    alien -i oracle-instantclient12.2-sqlplus-12.2.0.1.0-1.x86_64.rpm &&\
    alien -i oracle-instantclient12.2-tools-12.2.0.1.0-1.x86_64.rpm &&\
    ln -s /usr/include/oracle/12.2/client64 ${ORACLE_HOME}/include && \
    rm -f oracle*.rpm

WORKDIR build
COPY . .

RUN wget -O - https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab-linux-amd64-install.sh | sh && \
    buildfab install-version && \
    buildfab platform-view -v

#RUN --mount=type=bind,source=.cache/debian12/.conan2,target=$CONAN_PATH,rw

RUN conan profile detect --force && \
    conan remote add conan-local http://$CONAN_ARTIFACTORY/artifactory/api/conan/conan-local && \
    conan remote add conan-proxy http://$CONAN_ARTIFACTORY/artifactory/api/conan/conan-proxy && \
    conan remote disable conancenter

RUN mkdir .build && \
    export LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:${ORACLE_HOME}/lib && \
    buildfab build -vv && \
    strip -s bin/imsi-region

################################################################################

FROM debian:12.7-slim as runtime

ENV ORACLE_HOME=/usr/lib/oracle/12.2/client64
ARG VERSION

LABEL org.opencontainers.image.vendor="LCC Svyazcom"
LABEL org.opencontainers.image.licence="proprietary"
LABEL org.opencontainers.image.author="Aleksey.Ozhigov<burnes@svyazcom.ru>"
LABEL org.opencontainers.image.source="git@gitsrv.svyazcom.ru:steering/imsi-region.git"
LABEL org.opencontainers.image.title="Location update steering module, imsi-region"
LABEL org.opencontainers.image.version="${VERSION}"

RUN apt-get update && \
    apt-get install -y libev-dev libpcre3 libgoogle-perftools4 libaio1

COPY --from=builder /build/bin/ /usr/local/bin/
COPY --from=builder /build/etc/ /usr/local/etc/
COPY --from=builder $ORACLE_HOME/lib $ORACLE_HOME/lib

RUN echo "$ORACLE_HOME/lib" > /etc/ld.so.conf.d/oracle.conf && \
    ldconfig

EXPOSE 80/tcp

CMD ["/usr/local/bin/imsi-region", "-c", "/usr/local/etc/imsi-region.conf", "-i", "127.0.0.1", "-q"]
