# Docker

`docker ps` list running containers ^run
`docker ps -a` list all containers ^run
`docker images` list images ^run
`docker build -t {{tag}} .` build image ^run:tag
`docker run -it {{image}} bash` run interactive ^run:image
`docker exec -it {{container}} bash` shell into container ^run:container
`docker logs -f {{container}}` follow logs ^run:container
`docker stop {{container}}` stop container ^run:container
`docker rm {{container}}` remove container ^run:container
`docker rmi {{image}}` remove image ^run:image
`docker compose up -d` start compose stack ^run
`docker compose down` stop compose stack ^run
`docker compose logs -f` follow compose logs ^run
`docker compose ps` list compose services ^run
`docker system prune -f` clean unused data ^run
`docker volume ls` list volumes ^run
`docker network ls` list networks ^run
`docker inspect {{container}}` inspect container ^run:container
`docker pull {{image}}` pull image ^run:image
`docker tag {{source}} {{target}}` tag image ^run:source
