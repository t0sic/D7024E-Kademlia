sudo docker compose build
sudo docker compose up -d --scale node=5
sudo docker compose logs -f node