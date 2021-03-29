#!/bin/bash

MONGODB1=mongo

echo "Waiting for MongoDB startup..."
until curl http://${MONGODB1}:27017/serverStatus\?text\=1 2>&1 | grep uptime | head -1; do
  printf '.'
  sleep 1
done

# check if replica set is already initiated
RS_STATUS=$( mongo --quiet --host ${MONGODB1}:27017 --eval "rs.status().ok" )
if [[ $RS_STATUS != 1 ]]
then
  echo "[INFO] Replication set config invalid. Reconfiguring now."
  RS_CONFIG_STATUS=$( mongo --quiet --host ${MONGODB1}:27017 -u --eval "rs.status().codeName" )
  if [[ $RS_CONFIG_STATUS == 'InvalidReplicaSetConfig' ]]
  then
    mongo --quiet --host ${MONGODB1}:27017 <<EOF
config = rs.config()
config.members[0].host = 'localhost:27017' # Here is important to set the host name of the DB instance
rs.reconfig(config, {force: true})
EOF
  else
    echo "[INFO] MongoDB setup finished. Initiating replicata set."
    mongo --quiet --host ${MONGODB1}:27017 --eval "rs.initiate({_id:'rs0', members:[{_id:0, host:'localhost:27017'}]})" > /dev/null
  fi
else
  echo "[INFO] Replication set already initiated."
fi

