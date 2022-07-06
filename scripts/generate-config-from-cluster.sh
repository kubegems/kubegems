#! /bin/sh

argocdPassword=$(kubectl -n kubegems get secrets argocd-secret -o jsonpath='{.data.clearPassword}' | base64 -d)
redisPassword=$(kubectl -n kubegems get secrets kubegems-redis -ogo-template='{{ index .data "redis-password" | base64decode }}')
giteaPassword=$(kubectl -n kubegems get secrets kubegems-config -ogo-template='{{ .data.GIT_PASSWORD | base64decode }}')
giteaUsername=$(kubectl -n kubegems get secrets kubegems-config -ogo-template='{{ .data.GIT_USERNAME | base64decode }}')
mysqlPassword=$(kubectl -n kubegems get secrets kubegems-mysql -ogo-template='{{ index .data "mysql-root-password" | base64decode }}')
cat <<EOF | tee config/config.yaml
mysql:
  password: ${mysqlPassword}
redis:
  password: ${redisPassword}
git:
  username: ${giteaUsername}
  password: ${giteaPassword}
argo:
  password: ${argocdPassword}
EOF
