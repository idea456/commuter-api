dev-cluster:
	kind create cluster --config=manifests/cluster.dev.yml

setup-dev:
	kind load docker-image idea456/myjejak-planner-api && kubectl create -f manifests/pods.yml

destroy-all:
	kubectl delete deployment --all && kubectl delete svc --all

db-dev:
	docker run \
    --restart always \
	--env NEO4J_PLUGINS='["graph-data-science"]' \
    --publish=7474:7474 --publish=7687:7687 \
    neo4j:5.22.0

dev:
	go run cmd/api/main.go