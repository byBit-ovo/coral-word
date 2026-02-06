# Nginx 接入说明

将 Nginx 作为反向代理，对外提供 80 端口，将请求转发到本项目的 Go Gin 服务。

## 1. 安装 Nginx

**Ubuntu / Debian：**
```bash
sudo apt update
sudo apt install nginx
```

**CentOS / RHEL：**
```bash
sudo yum install nginx
# 或
sudo dnf install nginx
```

## 2. 启用站点配置

把本目录下的配置链接到 Nginx 站点目录（**请把路径换成你本机的 coral_word 项目路径**）：

```bash
# Ubuntu / Debian
sudo ln -s /path/to/coral_word/nginx/coral_word.conf /etc/nginx/sites-enabled/

# 如使用 sites-available 且需单独建链接
sudo ln -s /etc/nginx/sites-available/coral_word.conf /etc/nginx/sites-enabled/
# 然后把本项目的 coral_word.conf 复制到 sites-available，或直接：
sudo ln -s /home/qzr/gitee/coral_word/nginx/coral_word.conf /etc/nginx/sites-enabled/
```

**CentOS / RHEL** 通常把配置放在 `/etc/nginx/conf.d/`：
```bash
sudo ln -s /path/to/coral_word/nginx/coral_word.conf /etc/nginx/conf.d/coral_word.conf
```

**Ubuntu / Debian 务必禁用系统默认站点**，否则会报 `duplicate default server` 且用 IP 访问会落到默认页：
```bash
sudo unlink /etc/nginx/sites-enabled/default
```

## 3. 启动 Go 服务

确保 Gin 监听 **8080**（与 `coral_word.conf` 里 `proxy_pass` 一致）：

```bash
cd /path/to/coral_word
HTTP_ADDR=0.0.0.0:8080 go run .
# 或使用 .env：HTTP_ADDR=0.0.0.0:8080
```

## 4. 检查并重载 Nginx

```bash
sudo nginx -t
sudo systemctl reload nginx
```

## 5. 确认 80 端口在监听

```bash
# 只看 TCP 监听端口（推荐）
ss -tlnp | grep :80
# 或
sudo netstat -tlnp | grep :80
```

应能看到类似 `0.0.0.0:80`、`:::80` 且进程为 nginx。若没有，说明当前加载的配置里没有 `listen 80`，请确认：
- `coral_word.conf` 已在 `/etc/nginx/sites-enabled/` 下（或已正确 include）；
- 执行 `sudo nginx -T 2>/dev/null | grep -E 'listen|server_name'` 能看到 coral_word 的 `listen 80`。

## 6. 访问

- 浏览器打开：`http://localhost` 或 `http://你的服务器IP`
- 流量经 Nginx(80) → Gin(8080)，API 与页面与直接访问 8080 一致

---

## 特定接口走特定路由

用 **`location`** 按路径分流，不同 path 转发到不同后端或做不同处理。

| 写法 | 含义 | 示例 |
|------|------|------|
| `location = /path` | 精确匹配 | `/word` 只匹配 `/word`（含 query） |
| `location /prefix/` | 前缀匹配 | `/api/` 匹配 `/api/xxx` |
| `location ~ \.php$` | 正则（区分大小写） | 以 `.php` 结尾 |
| `location ~* \.(jpg\|png)$` | 正则（不区分大小写） | 图片后缀 |

**规则**：更具体的写在上面；同一请求只会进一个 location（匹配优先级：精确 > 最长前缀 > 正则 > 默认前缀）。  

示例见 `coral_word.conf` 内注释：只把 `/word` 转 8080、`/api/` 转 9090、`/static/` 走本地目录等，按需取消注释并改端口/路径后 `sudo nginx -t && sudo systemctl reload nginx`。

---

## 并发超不过 ~800 的原因与调优

**结论：多数情况下是 Nginx 默认连接数上限，不是 coral_word 本身。**

- Nginx 默认 `worker_connections` 多为 **768**（部分发行版 1024），单 worker 时最大并发约 768～1024，所以会卡在 ~800。
- 调大方式：改**主配置** `/etc/nginx/nginx.conf`（不是 sites-enabled 里的 server）：
  1. 在 **`events { }`** 里提高连接数，例如：
     ```nginx
     events {
         worker_connections 4096;   # 或 8192，按需调整
     }
     ```
  2. **`worker_processes`** 保持或设为 `auto`（按 CPU 核数），总并发 ≈ `worker_processes × worker_connections`。
  3. 保存后执行 `sudo nginx -t && sudo systemctl reload nginx`。

**coral_word 自身的限制（Nginx 调高后可能成为新瓶颈）：**

| 组件 | 当前限制 | 位置 |
|------|----------|------|
| MySQL | MaxOpenConns 100 | `sql.go` |
| LLM 协程池 | 10 worker + 200 队列 | `main.go` LLMPool |
| ES 客户端 | MaxIdleConnsPerHost 10 | `es.go` |

若要继续提高并发，可适当调大上述参数（并配合 DB/Redis/ES 与 LLM 上游能力）。

## 可选：静态文件由 Nginx 提供

若希望 CSS/JS 等由 Nginx 直接提供，可编辑 `coral_word.conf`：

1. 取消注释 `location /static/` 段；
2. 将 `CORAL_WORD_ROOT` 替换为项目根目录的绝对路径（如 `/home/qzr/gitee/coral_word`）。

保存后执行 `sudo nginx -t && sudo systemctl reload nginx`。

---

## 故障排除

### 浏览器访问到的是 Apache 默认页

说明 **80 端口被 Apache 占用**，请求没到 Nginx。停用 Apache 并启动 Nginx：

```bash
sudo systemctl stop apache2
sudo systemctl disable apache2   # 开机不自动启动（可选）
sudo systemctl start nginx
```

再访问 http://localhost 应看到 coral_word 的页面。

---

### `bind() to 0.0.0.0:80 failed (98)` 或 `still could not bind()`

表示 **80 端口已被占用**。先查占用进程：

```bash
sudo ss -tlnp | grep :80
# 或
sudo lsof -i :80
```

**处理方式：**

1. **停掉占用 80 的进程**（若是 Apache/其他 Web 服务）：
   ```bash
   sudo systemctl stop apache2   # 若为 Apache
   sudo systemctl start nginx
   ```

2. **或让 Nginx 改用其他端口**（如 8888）：  
   编辑 `coral_word.conf`，把 `listen 80;` 改为 `listen 8888;`，保存后：
   ```bash
   sudo nginx -t
   sudo systemctl start nginx
   ```
   访问时用 `http://localhost:8888`。
