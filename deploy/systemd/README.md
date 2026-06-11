# deploy/systemd/

手工部署用的 systemd unit：Go API 与 Next.js web 各一个常驻服务。
（PostgreSQL / Neo4j 不在此处管理，按你已有方式部署。）

## 服务器目录约定

unit 文件按 `/opt/matrix` 为根编写，按需改 `WorkingDirectory` / `ExecStart`：

```
/opt/matrix/
├── api/
│   ├── server              # go build 产出的二进制（即仓库 bin/server）
│   └── configs/
│       └── config.yaml     # 生产配置：host 改实际地址、secure_cookie:true、回调改线上域名
└── web/
    ├── .next/  node_modules/  public/  package.json   # next build 后的完整产物
```

## 一、准备产物

```bash
# 本机或服务器上构建
make build-api            # 产出 bin/server
make build-web            # 产出 apps/web/.next

# 拷到服务器（示例）
scp bin/server                 server:/opt/matrix/api/server
scp -r apps/api/configs        server:/opt/matrix/api/configs
scp -r apps/web/{.next,public,package.json,node_modules}  server:/opt/matrix/web/
```

> web 也可直接在服务器 `git pull && pnpm install && pnpm build`，省去拷 node_modules。

## 二、系统用户与密钥

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin matrix
sudo chown -R matrix:matrix /opt/matrix

sudo mkdir -p /etc/matrix
sudo cp deploy/systemd/api.env.example /etc/matrix/api.env
sudo cp deploy/systemd/web.env.example /etc/matrix/web.env
sudo chmod 600 /etc/matrix/*.env
sudo chown matrix:matrix /etc/matrix/*.env
# 编辑两个 env，填入真实值（MATRIX_CRYPTO_KEY、INTERNAL_SECRET 等）
```

## 三、安装并启动

```bash
sudo cp deploy/systemd/matrix-api.service /etc/systemd/system/
sudo cp deploy/systemd/matrix-web.service /etc/systemd/system/
sudo systemctl daemon-reload

sudo systemctl enable --now matrix-api
sudo systemctl enable --now matrix-web

# 查看状态 / 日志
systemctl status matrix-api matrix-web
journalctl -u matrix-api -f
```

## 注意

- `which node` 确认 node 路径，若不是 `/usr/bin/node` 改 `matrix-web.service` 的 ExecStart。
- API 的 `MATRIX_CRYPTO_KEY` 必填且 64 位 hex，否则启动 panic。
- 改完 unit 或 env 后：`sudo systemctl daemon-reload && sudo systemctl restart matrix-api`。
- 对外 HTTPS / 域名分流请在前面再放一层 nginx 或 Caddy（本目录不含）。
