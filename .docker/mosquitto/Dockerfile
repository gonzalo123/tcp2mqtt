FROM eclipse-mosquitto

ADD mosquitto.conf /mosquitto/config/mosquitto.conf
ADD users.txt /mosquitto/config/users.txt

WORKDIR /src

EXPOSE 1883
EXPOSE 9001
