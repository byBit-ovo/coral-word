ab -n 1000 -c 100 http://127.0.0.1/word?word=hello


# 查看当前限制（soft软限制，hard硬限制）
ulimit -n  # 输出当前软限制，比如24/1024

# 临时提升软限制（比如调到2000，不超过hard限制）
ulimit -n 2000

