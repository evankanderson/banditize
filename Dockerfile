FROM ghcr.io/pycqa/bandit/bandit@sha256:82b81adc7ac8394e35da72fd34eb56a5d56f8a32bfec7bf1b8ad9188a840ac89

# TODO: this is naughty!  I'm running `GOOS=linux go build .` on my mac, so the binary is built outside the dockerfile
ADD ./banditize /banditize
ENTRYPOINT [ "/banditize" ]