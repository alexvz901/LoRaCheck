# LoRaCheck

LoRaCheck is a dashboard made to monitor the uptime of IOT gateways.
It was made by three students as an assignment for school.

# Prerequisites
A [Docker](https://www.docker.com/) installation.

# Installation

To install LoRaCheck to docker:
1. download all the files in the repository.
2. extract the files to a chosen folder.
3. in a docker terminal, open the 'src' directory.
4. execute the command 'docker-compose up'.
5. open the dashboard on 'localhost:3000'.

# How to add gateways to the dashboard
![afbeelding](https://github.com/user-attachments/assets/bf831941-c939-4b30-9d85-80cd79f8aed7)
To add new gateways to the dashboard, you can upload them from the included website.
You will need to input the following information:
1. a name for the gateway
2. gateway location (latitude and longitude)
3. gateway link (http(s) or api fetch)
After this you can add them to the server and view them in the dashboard.
