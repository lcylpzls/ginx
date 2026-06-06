#!/bin/sh
#
# ginx — TLS 自签名证书一键生成脚本 (POSIX sh 兼容)
#
# 用法:
#   sh gen_cert.sh <证书输出目录> [IP/域名...]
#
# 示例:
#   sh gen_cert.sh ./certs                               # 仅 localhost
#   sh gen_cert.sh ./certs 192.168.1.100 app.local       # 指定 IP 和域名
#
# 产物:
#   ca-cert.pem       CA 根证书（公钥）→ 分发到所有客户端
#   ca-key.pem        CA 私钥        → 仅服务端，严格保密
#   server-cert.pem   Server 证书（含 SAN） → 仅服务端
#   server-key.pem    Server 私钥    → 仅服务端
#
set -eu

# ── 参数解析：第一个参数必须为证书输出目录 ──
CERT_DIR=""
SAN_ENTRIES="DNS:localhost,IP:127.0.0.1"

case "${1:-}" in
    ""|--help|-h)
        printf "用法: sh gen_cert.sh <证书输出目录> [IP/域名...]\n"
        printf "\n"
        printf "示例:\n"
        printf "  sh gen_cert.sh ./certs\n"
        printf "  sh gen_cert.sh ./certs 192.168.1.100 app.local\n"
        printf "\n"
        printf "第一个参数（证书输出目录）为必填。\n"
        exit 1
        ;;
    *)
        CERT_DIR="$1"
        shift
        ;;
esac

# 剩余参数作为 SAN (Subject Alternative Name)
for addr in "$@"; do
    case "$addr" in
        *[!0-9.]*) SAN_ENTRIES="$SAN_ENTRIES,DNS:$addr" ;;
        *)         SAN_ENTRIES="$SAN_ENTRIES,IP:$addr" ;;
    esac
done

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# detect_pkg_install 根据当前系统包管理器返回 openssl 的安装命令。
detect_pkg_install() {
    if command -v apt-get >/dev/null 2>&1; then
        printf "apt-get install openssl"
    elif command -v dnf >/dev/null 2>&1; then
        printf "dnf install openssl"
    elif command -v yum >/dev/null 2>&1; then
        printf "yum install openssl"
    elif command -v apk >/dev/null 2>&1; then
        printf "apk add openssl"
    elif command -v pacman >/dev/null 2>&1; then
        printf "pacman -S openssl"
    elif command -v zypper >/dev/null 2>&1; then
        printf "zypper install openssl"
    elif command -v brew >/dev/null 2>&1; then
        printf "brew install openssl"
    else
        printf "请通过系统包管理器安装 openssl"
    fi
}

if ! command -v openssl >/dev/null 2>&1; then
    printf "${RED}[错误] 未找到 openssl，请先安装: %s${NC}\n" "$(detect_pkg_install)"
    exit 1
fi

mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

printf "============================================\n"
printf "  ginx — TLS 证书生成\n"
printf "  输出目录: %s\n" "$CERT_DIR"
printf "============================================\n"
printf "\n"

# ── 1. 生成 CA 私钥 ──
printf "[1/4] 生成 CA 私钥 (ec-p256)...\n"
openssl ecparam -genkey -name prime256v1 -noout -out ca-key.pem

# ── 2. 生成 CA 根证书（自签名，10年有效） ──
printf "[2/4] 生成 CA 根证书 (自签名, 10年)...\n"
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem \
    -subj "/CN=ginx Internal CA/O=ginx"

# ── 3. 生成 Server 私钥 ──
printf "[3/4] 生成 Server 私钥 (ec-p256)...\n"
openssl ecparam -genkey -name prime256v1 -noout -out server-key.pem

# ── 4. 生成 Server 证书（CA 签发，1年有效，含 SAN） ──
printf "[4/4] 生成 Server 证书 (CA 签发, 1年, SAN: %s)...\n" "$SAN_ENTRIES"

printf "subjectAltName = %s\n" "$SAN_ENTRIES" > server.ext

openssl req -new -key server-key.pem -out server.csr \
    -subj "/CN=ginx-server/O=ginx"

openssl x509 -req -days 365 -in server.csr -CA ca-cert.pem -CAkey ca-key.pem \
    -CAcreateserial -out server-cert.pem -extfile server.ext

# 清理临时文件
rm -f server.csr server.ext

# ── 设置权限 ──
chmod 600 ca-key.pem server-key.pem
chmod 644 ca-cert.pem server-cert.pem

printf "\n"
printf "${GREEN}============================================${NC}\n"
printf "${GREEN}  证书生成完成!${NC}\n"
printf "${GREEN}============================================${NC}\n"
printf "\n"
printf "产物列表:\n"
ls -la "$CERT_DIR"/*.pem
printf "\n"
printf "${YELLOW}部署提醒:${NC}\n"
printf "  1. 将 %s/ca-cert.pem 分发到所有客户端机器\n" "$CERT_DIR"
printf "     scp %s/ca-cert.pem user@client:/path/to/certs/\n" "$CERT_DIR"
printf "\n"
printf "  2. ca-key.pem 和 server-key.pem 严格保密，不可外传\n"
printf "\n"
printf "  3. ginx 配置示例:\n"
printf '     ginx.Config{TLSCertFile: "%s/server-cert.pem", TLSKeyFile: "%s/server-key.pem"}\n' \
    "$CERT_DIR" "$CERT_DIR"
