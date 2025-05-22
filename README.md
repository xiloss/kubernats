# KUBERNATS

Let's run NATS on kubernetes!

Current Prerequisites:
- golang 1.22
- kubebuilder 4.2.0
- a kubernetes cluster, the current demo is using a 3 nodes local kind, based on version 1.27.3 of kubernetes

## MAINTAINERS AND CODE CONTRIBUTORS

In order to setup your local environment to contribute the kubernats operator project,
please be sure to follow the next steps

### How it has been created (section for learning purpose)

#### Kubebuilder Setup

First the kubebuilder setup:

```bashwebapp-operator
# Go to your home directory
cd ~
# Current kubebuilder latest version
KUBEBUILDER_VERSION="4.2.0"
# Kubebuilder github release endpoint
KUBEBUILDER_URL="https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}"
# Download the version of kubebuilder 4.2.0 for this demo
curl -L ${KUBEBUILDER_URL}/kubebuilder_linux_amd64 -o kubebuilder-${KUBEBUILDER_VERSION}
# Make the archive executable
chmod ugo+x kubebuilder
# Move the extracted binary to a permanent location
sudo mv kubebuilder /usr/local/bin/kubebuilder-${KUBEBUILDER_VERSION}
# ensure you copy the executable versioned, in order to persist various versions
sudo ln -sfn /usr/local/bin/kubebuilder-${KUBEBUILDER_VERSION} /usr/local/bin/kubebuilder
```

#### Kubebuilder Initialization

Everything is now ready to initialize the kubebuilder project.
The first thing to do is to assign the domain name to our API extension.
This could be whatever we want to use as the main domain name for our components.
Please note that it is not necessary an existing domain.
If you pre-scaffolded the project, please be sure to remove an eventually existing main.go file.

NOTE: existing go.mod files will be overwritten.

```bash
# Initialize Kubebuilder project
kubebuilder init \
  --domain kubernats.ai \
  --owner xiloss \
  --project-name kubernats \
  --repo github.com/xiloss/kubernats \
  --project-version 3 \
  --plugins go/v4
```

#### Creating the JetStream API

```bash
# Create Kubebuilder api for kind JetStream
# using yes we will not be asked for confirmation, creating the resources in unattended mode
yes | kubebuilder create api --group apps --version v1alpha1 --kind JetStream
```

#### Creating the AuthSecret API

```bash
# Create Kubebuilder api for kind AuthSecret
# using yes we will not be asked for confirmation, creating the resources in unattended mode
yes | kubebuilder create api --group apps --version v1alpha1 --kind AuthSecret
```

#### Creating the Consumer API

```bash
# Create Kubebuilder api for kind Consumer
# using yes we will not be asked for confirmation, creating the resources in unattended mode
yes | kubebuilder create api --group apps --version v1alpha1 --kind Consumer
```

#### Creating the KeyValue Store API

```bash
# Create Kubebuilder api for kind KeyValueStore
# using yes we will not be asked for confirmation, creating the resources in unattended mode
yes | kubebuilder create api --group apps --version v1alpha1 --kind KeyValueStore
```
