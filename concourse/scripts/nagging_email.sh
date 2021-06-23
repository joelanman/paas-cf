#!/bin/sh

set -eu

DEPLOY_ENV=$1
SYSTEM_DNS_ZONE_NAME=$2
ALERT_EMAIL_ADDRESS=$3
MESSAGE_TYPE=$4

TO="${ALERT_EMAIL_ADDRESS}"
FROM="${ALERT_EMAIL_ADDRESS}"

write_message_json() {
  if [ "${MESSAGE_TYPE}" = 'resurrector-disabled' ]; then
    cat <<EOF > message.json
{
  "Subject": {
    "Data": "Resurrector is disabled in ${DEPLOY_ENV}"
  },
  "Body": {
    "Html": {
      "Data": "Bosh's resurrector is currently disabled in <b>${DEPLOY_ENV}</b>. Presumably \
      this is deliberate, but you probably want to re-enable it as soon as you deem sensible, \
      to avoid having to go and recreate dead instances manually. See \
      <a href='https://deployer.${SYSTEM_DNS_ZONE_NAME}/teams/main/pipelines/create-cloudfoundry?group=health'>Concourse</a> \
      for details<br/>Alternatively, something else caused this check to fail, which is also \
      something that should be investigated."
    }
  }
}
EOF
  fi
}

write_message_json

aws ses send-email --region eu-west-1 --to "${TO}" --message file://message.json --from "${FROM}"
