# This is the base image of the services.
# In this stage we prepare all required common data to dockerize our services.
FROM golang:1.19

ARG BIN
ENV CMD_DIR ${BIN}
#ARG GITHUB_TOKEN

#RUN apt-get update                                                        && \
#  DEBIAN_FRONTEND=noninteractive apt-get install -yq --no-install-recommends \
#    curl                                                                     \
#    git                                                                      \
#    unzip                                                                    \
#  && rm -rf /var/lib/apt/lists/*


# SSH key for git
#ADD /home/${USER}/.ssh/id_rsa /root/.ssh/id_rsa
#RUN chmod 400 /root/.ssh/id_rsa
#RUN echo "Host github.com\n\tStrictHostKeyChecking no\n" >> /root/.ssh/config
#
#
#COPY github_key .
#RUN git config --global --add url."git@github.com:".insteadOf "https://github.com/"
#RUN eval $(ssh-agent) && \
#    ssh-add github_key && \
#    ssh-keyscan -H github.com >> /etc/ssh/ssh_known_hosts && \
#    git clone git@github.com:dpmx/mtjob-go.git /mtjob

#RUN git clone https://${GITHUB_TOKEN}@github.com/dpmx/mtjob-go.git /mtjob
WORKDIR /tracking-fb
COPY . .

ENV GO111MODULE=on

RUN make native BIN=$CMD_DIR

RUN mkdir /app && cp -r dist/$CMD_DIR/* /app
#COPY conf_docker.toml /app/conf.toml

# expose port for api
EXPOSE 8888
# expose port for dms
EXPOSE 8899

WORKDIR /app

ENTRYPOINT ["./run.sh", "start"]