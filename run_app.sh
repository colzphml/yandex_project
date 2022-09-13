cd ./cmd/agent/
sh ./build_container.sh
cd ./../server/
sh ./build_container.sh
cd ../../
docker compose up -d