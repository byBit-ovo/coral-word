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

## 5. 访问

- 浏览器打开：`http://localhost` 或 `http://你的服务器IP`
- 流量经 Nginx(80) → Gin(8080)，API 与页面与直接访问 8080 一致

## 可选：静态文件由 Nginx 提供

若希望 CSS/JS 等由 Nginx 直接提供，可编辑 `coral_word.conf`：

1. 取消注释 `location /static/` 段；
2. 将 `CORAL_WORD_ROOT` 替换为项目根目录的绝对路径（如 `/home/qzr/gitee/coral_word`）。

保存后执行 `sudo nginx -t && sudo systemctl reload nginx`。
