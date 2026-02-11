# CLIProxyAPI 二次开发 / 部署说明（本仓库约定）

目标：对外只暴露一个（或少量）`api-keys` 作为统一鉴权；对内在 `gemini-api-key`（以及其它 provider）里配置多 Key，实现轮询/重试/故障切换。TTS 也走同一套机制。

## Go 版本

- 本项目 `go.mod` 目前声明为 Go `1.24.0`（建议使用 Go `>= 1.24`）。

## 本地测试

- 已安装 Go：在 `CLIProxyAPI/` 下运行 `go test ./...`
- 未安装 Go（无 sudo 安装到用户目录示例）：
  - 下载并解压（以 Linux amd64 + go1.26.0 为例）：
    - `mkdir -p ~/.local/opt && cd /tmp && curl -fsSL https://mirrors.aliyun.com/golang/go1.26.0.linux-amd64.tar.gz -o go.tgz && tar -xzf go.tgz && rm -rf ~/.local/opt/go1.26.0 && mv go ~/.local/opt/go1.26.0 && ln -sfn ~/.local/opt/go1.26.0 ~/.local/opt/go`
  - 临时使用：
    - `~/.local/opt/go/bin/go test ./...`
  - 或加入 PATH（例如写入 `~/.bashrc`）：
    - `export PATH="$HOME/.local/opt/go/bin:$PATH"`

## 本地运行

- `go run ./cmd/server -config ./config.yaml`

## 最小可用配置（Key 池轮询）

说明：
- `api-keys`：调用方使用的“统一 key”（你项目里只需要配这一份）
- `gemini-api-key`：CLIProxyAPI 内部轮询/切换的 Gemini 官方 Key 列表

最小示例（把 `${...}` 换成真实值）：

```yaml
host: "0.0.0.0"
port: 8317

# 调用方鉴权（统一 key）
api-keys:
  - "${PROXY_API_KEY}"

# 可选：生产建议关闭，避免把音频 base64 写进日志
request-log: false

quota-exceeded:
  switch-project: true
  switch-preview-model: true

routing:
  strategy: "round-robin"

gemini-api-key:
  - api-key: "${GEMINI_KEY_1}"
  - api-key: "${GEMINI_KEY_2}"
```

## 使用 `GEMINI_API_KEY` 快速测试（直连官方 Key）

CLIProxyAPI 不会自动把环境变量展开进 `config.yaml`，建议用 shell 生成一份临时配置：

```bash
export PROXY_API_KEY="proxy_key_for_your_app"
export GEMINI_API_KEY="AIzaSy..."

cat > ./config.yaml <<EOF
host: "127.0.0.1"
port: 8317
api-keys:
  - "${PROXY_API_KEY}"
request-log: false
gemini-api-key:
  - api-key: "${GEMINI_API_KEY}"
EOF

go run ./cmd/server -config ./config.yaml
```

## Gemini 原生 TTS 支持（本分支新增）

- 新增注册的 TTS 模型（用于路由与轮询）：
  - `gemini-2.5-flash-preview-tts`
  - `gemini-2.5-pro-preview-tts`
- TTS 请求兼容性处理：
  - 当检测到 TTS 请求时，会在转发到官方 Gemini API 前剔除 `tools/toolConfig/safetySettings`
  - 相关逻辑集中在 `internal/runtime/executor/gemini_tts_helpers.go`

### TTS 调用示例（走代理统一 key）

请求：
- URL：`http://<host>:8317/v1beta/models/gemini-2.5-flash-preview-tts:generateContent`
- 鉴权：`Authorization: Bearer <PROXY_API_KEY>`（或 `X-Api-Key: <PROXY_API_KEY>`）

```bash
curl -sS "http://127.0.0.1:8317/v1beta/models/gemini-2.5-flash-preview-tts:generateContent" \
  -H "Authorization: Bearer ${PROXY_API_KEY}" \
  -H "Content-Type: application/json" \
  -d @- <<'JSON'
{
  "contents": [{"role": "user", "parts": [{"text": "你好，请用中文朗读这句话。"}]}],
  "generationConfig": {
    "responseModalities": ["AUDIO"],
    "speechConfig": {
      "voiceConfig": { "prebuiltVoiceConfig": { "voiceName": "Aoede" } }
    }
  }
}
JSON
```

备注：
- 响应音频一般在 `candidates[0].content.parts[0].inlineData`（base64），体积可能较大（尤其长文本）。
- Gemini 执行器对非流式响应会 `ReadAll`（单请求内存占用与音频长度正相关）；并发高 + 长音频时建议压测与限流。

## Web 管理页面（配置/查看用量/更新 config）

- 页面入口：`http://<host>:8317/management.html`
- 管理 API：`/v0/management/*`
- 启用条件（二选一即可）：
  - `remote-management.secret-key` 非空（可写明文，首次启动会自动 bcrypt 并回写 config）
  - 或设置环境变量 `MANAGEMENT_PASSWORD`（无需改 config）
- 鉴权方式：
  - `Authorization: Bearer <管理key>` 或 `X-Management-Key: <管理key>`
- 远程访问：
  - 默认 `remote-management.allow-remote: false`（仅 localhost 可访问管理 API）
  - 线上建议保持关闭，用 SSH 隧道访问：`ssh -L 8317:127.0.0.1:8317 user@server`

## 部署（不使用 Docker）

1) 编译：`go build -o cliproxy ./cmd/server`
2) 准备 `config.yaml`（同目录或指定 `-config`）
3) 运行：`./cliproxy -config ./config.yaml`

建议：
- 用 systemd 托管进程，配合 `Restart=always` + 日志轮转
- `host` 只在内网使用可设 `127.0.0.1`，再由 Nginx/Caddy 做 TLS 与鉴权

## 部署（Docker / docker compose）

### docker compose（推荐）

1) 复制配置：`cp config.example.yaml config.yaml` 并按需修改
2) 启动：`docker compose up -d --build`

关键挂载/端口（见 `docker-compose.yml`）：
- `./config.yaml:/CLIProxyAPI/config.yaml`
- `./auths:/root/.cli-proxy-api`（OAuth/token 文件）
- `./logs:/CLIProxyAPI/logs`
- 端口：`8317` 为主服务端口；其余映射端口主要用于部分 OAuth 回调/本地辅助流程（不用可不暴露）

### 管理页面在 Docker 下的注意事项

- Docker bridge + 端口映射时，容器内看到的客户端 IP 往往不是 `127.0.0.1`，因此：
  - 若要从宿主机访问管理页面/管理 API：建议直接设置 `MANAGEMENT_PASSWORD` 环境变量（会自动放开 `allow-remote` 逻辑，但仍要求管理 key）
  - 或在 `config.yaml` 设置 `remote-management.allow-remote: true`（强烈建议同时配防火墙，避免暴露到公网）

### Docker 是否必要 / 资源开销

- Docker 不是必需：二进制 + systemd 的方式同样适合服务器部署。
- 资源开销没有固定值，主要取决于你的并发与上游响应大小；容器本身额外开销通常较小，建议用 `docker stats` 实测。

## 调用方请求 URL 速查

- OpenAI 兼容：
  - `POST http://<host>:8317/v1/chat/completions`
  - `POST http://<host>:8317/v1/responses`
- Gemini 兼容：
  - `POST http://<host>:8317/v1beta/models/<model>:generateContent`
  - `POST http://<host>:8317/v1beta/models/<model>:streamGenerateContent`

调用方鉴权 header（择一）：
- `Authorization: Bearer <PROXY_API_KEY>`
- `X-Api-Key: <PROXY_API_KEY>`
- `X-Goog-Api-Key: <PROXY_API_KEY>`
