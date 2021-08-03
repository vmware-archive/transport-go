#!/bin/bash
AUTH_TARGET="https://$REGISTRY_PREFIX"
echo "Logging into $REGISTRY_PREFIX..."
if [ -z $CI ] ; then
    echo "This shell script is strictly to be used in the GitLab CI environment."
    echo "If you use the script in your local environment, it may accidentally overwrite your Docker config file"
    exit 1
fi

jq --version >/dev/null 2>&1

if [ $? -gt 0 ] ; then
    echo "jq is missing!" >&2
    exit 1
fi

mkdir -p ~/.docker
echo '{}' > $HOME/.docker/config.json
cat $HOME/.docker/config.json | jq ".auths[\"${AUTH_TARGET}\"] += {\"auth\": \"${DOCKER_REPO_AUTH_KEY}\", \"email\": \"${DOCKER_REPO_USERNAME}@vmware.com\"}" > $HOME/.docker/config.json.swp
mv $HOME/.docker/config.json.swp $HOME/.docker/config.json
docker login https://$REGISTRY_PREFIX

