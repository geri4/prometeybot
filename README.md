# Telegram bot for Prometheus monitoring system

This is simple service that takes alerts from Prometheus Alertmanger and sends it to Telegram chat.

Features:
* Authentication for receiveing alerts from bot
* Poll model for simple NAT traversal 

Prometeybot starts at port 9010 and listens events from Alertmanager.

Startup arguments:
```
  -apikey #Api key that you took from Telegram Bot Father (Required)
  -chatpassword #Password for access to telegram bot (Required)
  -dbpath #Path to telegram bot database (Not requried, default: /data/chat.db)
```
Or use system envs:
  - APIKEY
  - CHATPASSWORD
  - DBPATH


## Alermanager integration

Example of alertmanager config that use prometeybot:
```
receivers:
  - name: default
    webhook_configs:
    - send_resolved: True
      url: http://prometeybot:9010/sendalert
```