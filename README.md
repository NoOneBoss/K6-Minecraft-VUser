# K6-MinecraftVUser
**Расширение для k6, позволяющее создавать ботов в Minecraft для нагрузочного тестирования серверов**

[![xk6](https://img.shields.io/badge/build%20with-xk6-%23FF6C37)](https://github.com/grafana/xk6)

## Особенности
- Подключение к Minecraft-серверам (1.20.2)
- Обработка игровых событий (чат, смерть, здоровье)
- Отправка сообщений в чат
- Управление базовыми действиями бота
- Интеграция с k6-скриптами

## Установка
### Сборка с Docker
1. Создайте Dockerfile:
   ```dockerfile
   FROM golang:1.24 as builder

   RUN go install go.k6.io/xk6/cmd/xk6@latest
   
   RUN xk6 build --with github.com/NoOneBoss/K6-MinecraftVUser@latest
   
   FROM alpine:3.18
   COPY --from=builder /go/k6 /usr/bin/k6
   ENTRYPOINT ["k6"]
   ```

2. Соберите образ:
    ```
    docker build -t k6-minecraft .
    ```
## Использование
### Пример теста (test.js)
   ```
      import minecraft from 'k6/x/minecraft';
      import { sleep } from 'k6'
      
      export default function () {
          const bot = minecraft.newBot();
      
          bot.connect('localhost:25565', 'K6Bot', '', '')
      
          if (!bot.waitForHealth(5000)) {
              throw new Error('Health update timeout');
          }
          const health = bot.getHealth();
          console.log(`Health: ${health}`);
      
          bot.sendMessage('Test from k6');
      
          if (!bot.waitForMessage(1000)) {
              throw new Error('Message timeout');
          }
          sleep(2)
          const lastMsg = bot.getLastMessage();
          console.log(`Last message: ${lastMsg}`);
      }
   ```

## Ограничения
- Поддерживается только offline-режим серверов
- Античит/auth системы могут блокировать ботов
