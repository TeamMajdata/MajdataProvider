# Majdata Provider

> This implementation is based on the official Online Server API Guide:  
> 本项目实现基于官方 Online Server API 文档规范：  
> https://github.com/LingFeng-bbben/MajdataPlay/wiki/Online-Server-API-Guide

**Provide your own charts, make them available online now!**  
使用 **Majdata Provider** 创建你自己的在线谱面源，并在 **MajdataPlay** 中添加该源，直接游玩在线谱面！

---

## 📦 What is Majdata Provider? 这是什么？

Majdata Provider is a lightweight HTTP server implemented in Go that follows the Online Server API and hosts your custom charts.  
Majdata Provider 是一个使用 Go 实现、遵循 Online Server API 规范的轻量级 HTTP 服务端程序，用来托管你自己的谱面资源。

You control 你可以完全掌控：
- Your own charts 谱面内容
- Your own update cycle 更新节奏
- Your own server 服务器部署方式
- Your own community 社区生态

---

## 🧱 Build Your Own Provider 构建你的 Provider

### 1️⃣ Get the code 拉取代码

```bash
git clone https://github.com/TeamMajdata/MajdataProvider.git
cd MajdataProvider
```

Build steps depend on your platform/toolchain. Please follow the repository instructions.  
编译方式与平台/工具链有关，请以仓库说明为准。

### 📁 Add Your Charts 放入你的谱面

Place your charts inside the `charts/` folder:  
把谱面资源放进 `charts/` 文件夹：

```text
charts/
```

#### 📌 Important 重要

- The `charts` folder should be placed under the same directory as the Provider executable. `charts` 文件夹应与 Provider 可执行文件处于同一路径（同级目录）。

#### ✅ Subfolders supported 支持子文件夹整理

You may create subfolders under `charts/` for organizing (e.g. by pack, artist, genre).  
你可以在 `charts/` 下创建任意层级的子文件夹用于整理（例如按曲包/曲师/分类）。

The provider will recursively scan the `charts/` directory and load charts inside subfolders automatically.  
Provider 会对 `charts/` 目录进行递归扫描，子文件夹里的谱面会被自动识别并加载。

Example 示例结构：

```text
MajdataProvider
charts/
├── Pop/
│   ├── SongA/
│   └── SongB/
├── Hardcore/
│   └── SongC/
└── Packs/
    ├── Pack01/
    │   ├── SongD/
    │   └── SongE/
    └── Pack02/
        └── SongF/
```

### 🚀 Run the Provider 运行服务

By default, Majdata Provider listens on:  
默认监听地址与端口：

```text
http://localhost:8080
```

If deploying to a server, you can test with:  
如果部署在服务器上，可通过以下方式访问测试：

```text
http://your-ip:8080
```

## 🌍 Deploy Online 部署到公网（正式环境）

To make your charts source accessible online, you typically need:  
想让别人能访问你的在线谱面源，一般需要：

- A VPS or public server（VPS 公网服务器）
- A static public IP (recommended)（推荐静态公网 IP）
- Firewall allows the port you use（防火墙放行端口）

Tip: You can keep the Provider on localhost and only expose HTTPS via reverse proxy.  
小贴士：可以让 Provider 仅监听本机，再通过反向代理对外提供 HTTPS。

### 🔒 Reverse Proxy with Caddy (Recommended) 用 Caddy 反向代理（推荐）

Instead of exposing port 8080 directly, use Caddy as a reverse proxy.  
不建议直接暴露 8080，推荐使用 Caddy 做反向代理。

Install Caddy 安装 Caddy（Ubuntu/Debian 示例）：

```bash
sudo apt update
sudo apt install -y caddy
```

Example Caddyfile 示例 Caddyfile：

Replace `yourdomain.com` with your domain.  
把 `yourdomain.com` 替换成你的域名。

```caddyfile
yourdomain.com {
  reverse_proxy 127.0.0.1:8080
}
```

Reload Caddy 重载 Caddy：

```bash
sudo systemctl reload caddy
```

Now your charts source should be available at:  
现在你的谱面源可以通过以下地址访问：

```text
https://yourdomain.com
```

Caddy will automatically obtain HTTPS certificates.  
Caddy 会自动签发并续期 HTTPS 证书。

## 🎮 Add Source in MajdataPlay 在 MajdataPlay 中添加源

Inside MajdataPlay 在 MajdataPlay 里：
1. Find the game data directory and open `settings.json`. 找到游戏数据目录并打开 `settings.json`。
2. Locate `Online.ApiEndPoints` in the file. 在文件中找到 `Online.ApiEndPoints` 配置项。
3. Add your provider URL (for example: `https://yourdomain.com`) to `ApiEndPoints`. 将你的 Provider 地址（例如：`https://yourdomain.com`）添加到 `ApiEndPoints`。
4. Minimal example 最小示例：

```json
{
  "Online": {
    "Enable": true,
    "ApiEndPoints": [{
      "Name": "MyChartSource",
      "Url": "https://yourdomain.com",
      "Username": "YourUsername",
      "Password": "YourPassword"
      }]
  }
}
```

5. Save `settings.json` and restart MajdataPlay. 保存 `settings.json` 并重启 MajdataPlay。
6. Refresh the charts source list in game. 回到游戏内刷新谱面源列表。

Then you can browse & play charts from your own online source.  
完成后即可浏览并游玩你的在线谱面。

## ⚠️ Notes 注意事项

- Make sure your server bandwidth is sufficient. 确保服务器带宽够用（谱面的音频或者视频资源可能较大）。
- Keep your folder structure consistent. 目录结构保持一致，便于维护。
- Consider caching or CDN if your users are global. 如果用户分布较广，可考虑缓存或 CDN 提升体验。

## 🔗 Related Projects 相关项目

- MajdataPlay: https://github.com/LingFeng-bbben/MajdataPlay

## ⚠️ Disclaimer 免责声明

### Copyright & Content 版权与内容

You are solely responsible for any content you host, share, or distribute using this server (including but not limited to music, images, charts, and metadata).  
你必须对你通过该服务器托管、分享或分发的任何内容（包括但不限于音乐、图片、谱面、元数据等）承担全部责任。

If you use copyrighted materials without permission, any legal consequences are your own responsibility.  
如果你未经授权使用受版权保护的资源，产生的一切法律后果由你自行承担。

This project’s contributors and maintainers do **not** provide any copyrighted resources and are **not** responsible for user-uploaded or user-hosted content.  
本项目贡献者/维护者 **不提供** 任何版权资源，也 **不对** 用户自行上传/托管的内容负责。

### No Warranty 不提供担保

This software is provided "AS IS", without warranty of any kind.  
本软件按“现状”提供，不提供任何形式的担保。

---

## 📜 License 许可证（GPL v3）

Majdata Provider is licensed under the **GNU General Public License v3 (GPL-3.0)**.  
Majdata Provider 使用 **GNU GPL v3 (GPL-3.0)** 许可证。

### What GPL v3 means (high-level) GPL v3 的基本要求（概览）

If you modify this program and distribute the modified version, you must:  
如果你对本程序进行修改并**对外分发**修改后的版本，你需要：

- Keep the same license (GPL v3) 仍然使用 GPL v3 许可证
- Provide the complete corresponding source code 提供完整的对应源代码
- Preserve copyright and license notices 保留版权与许可证声明
- State significant changes you made 标注你做过的重要修改
️
