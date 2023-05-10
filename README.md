# Introduction

Transactional Outbox Pattern in Go. Accompanies the blog post at https://www.gmhafiz.com/blog/transactional-outbox-pattern/

# Quick start

```sh
docker-compose up -d redis postgres mailpit asynqmon
```

Run main api server

```sh
go run cmd/api/main.go
```

Run workers (as many as you want)

```sh
go run cmd/worker/main.go
```

```
asynq: pid=209309 2023/02/20 11:32:46.998734 INFO: Starting processing
asynq: pid=209309 2023/02/20 11:32:46.998761 INFO: Send signal TSTP to stop processing new tasks
asynq: pid=209309 2023/02/20 11:32:46.998768 INFO: Send signal TERM or INT to terminate the process
```

Send email

```sh
curl -v http://localhost:3080/api/mail/send -H 'Content-type: application/json' -d '{"from":"from@example.com","to":["to@example.com"],"subject":"Test Subject","content":"some content"}'
```

# Monitor tasks

Go to http://localhost:8080 


# Show emails

Go to http://localhost:8025


# Simulate Errors

Simulate email sending error by turning off the email server and send an email

```sh
docker-compose stop mailpit

curl -v http://localhost:3080/api/mail/send -H 'Content-type: application/json' -d '{"from":"from@example.com","to":["to@example.com"],"subject":"Test Mail Server Is DOWN","content":"some content"}'

```

Watch the task is retrying for 25 times and its `mail_deliveries` records has both a `Failed` status and error message saved.

To resume, re-start the mail server

```sh
docker-compose up -d mailpit
```

Shut down all containers

```sh
docker-compose down
```

# Monitor Database

Using `pg_top`, we can see it is doing 1 transaction per second

```sh
sudo apt install pgtop
```

Run with

```sh
pgtop pg_top -h localhost  -p 5432 -d outbox_pattern -U user
```

```
last pid: 432794;  load avg  2.10,  2.17,  2.31;       up 1+14:17:12                                                                                                                                                10:01:27
5 processes: 5 sleeping
CPU states:  0.0% user,  3.7% nice,  1.0% system, 87.3% idle,  8.0% iowait
Memory: 54G used, 9089M free, 6822M buffers, 9511M cached
DB activity:   1 tps,  0 rollbs/s,   0 buffer r/s, 100 hit%,     66 row r/s,    0 row w/s 
DB I/O:     3 reads/s,   384 KB/s,   142 writes/s,   985 KB/s  
DB disk: 0.0 GB total, 0.0 GB free (100% used)
Swap: 385M used, 3710M free, 99M cached

  PID USERNAME PRI NICE  SIZE   RES STATE   TIME   WCPU    CPU COMMAND
   70 root      20    0    0K    0K sleep   0:01  0.02%  0.00% ksoftirqd/9
   27 root     -99    0    0K    0K sleep   0:00  0.00%  0.00% migration/2
   69 root     -99    0    0K    0K sleep   0:00  0.00%  0.00% migration/9
   31 root      20    0    0K    0K sleep   0:00  0.00%  0.00% cpuhp/3
   26 root     -51    0    0K    0K sleep   0:00  0.00%  0.00% idle_inject/2

```


# Graceful Shutdown

To prevent tasks from being dropped prematurely, we gracefully shut down each service properly. They
both listen to OS signals so if these were deployed in kubernetes, it will be handled automatically.
Otherwise, follow instructions in each section below.

## API

We know it is running at port `3080`

```sh
kill -SIGTERM $(lsof -t -i :3080) 
```

## Workers

Each worker has its own PID. Send a `TSTP` signal to stop processing new tasks, then send a `TERM` signal to shut down the worker

```sh
kill -SIGTSTP 209309
kill -SIGTERM 209309
```
