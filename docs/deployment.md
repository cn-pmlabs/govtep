docker load --input docker-sonic-vs.gz

docker run --name docker-sonic --hostname=sonic -it --privileged=true --network host -v /root/code:/code/ docker-sonic-vs

docker exec -it docker-sonic bash
