# genlog.py OUT: writes a 44 MB log of 600,000 lines, the same every time.
import random
import sys

random.seed(7)
levels = ["INFO", "DEBUG", "WARN", "ERROR"]
services = ["api", "auth", "db", "cache", "worker", "gateway"]
messages = [
    "request handled in %dms", "user %d logged in", "cache miss for key session:%d",
    "retrying connection attempt %d", "slow query took %dms: SELECT * FROM orders WHERE id = ?",
    "请求处理完成，耗时 %d 毫秒",
]
with open(sys.argv[1], "w") as f:
    for i in range(600000):
        f.write("2026-11-08T%02d:%02d:%02d.%03dZ %-5s [%s] %s\n" % (
            i // 36000 % 24, i // 600 % 60, i // 10 % 60, i % 1000,
            random.choice(levels), random.choice(services), random.choice(messages) % random.randint(1, 99999)))
