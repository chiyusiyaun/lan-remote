# LAN Remote

灞€鍩熺綉杩滅▼妗岄潰鎺у埗銆俉indows / Linux 浜掓帶锛屾墜鏈烘祻瑙堝櫒鍙綔涓绘帶绔€?
**涓嬭浇瀹夎鍖?鈫?[Releases](https://github.com/hyc-yuchen/lan-remote/releases/latest)**

```mermaid
graph LR
  subgraph S [lan-remote-server 缁熶竴绔彛]
    R[8765 娉ㄥ唽+闂ㄦ埛]
  end
  subgraph A [鐢佃剳 A client]
    CA[鎺у埗鍙?:8765 鎴睆+閿紶]
  end
  subgraph B [鐢佃剳 B client]
    CB[鎺у埗鍙?:8765 鎴睆+閿紶]
  end
  A -->|娉ㄥ唽/蹇冭烦| R
  B -->|娉ㄥ唽/蹇冭烦| R
  A <-->|WebSocket 鐢婚潰+杈撳叆| CB
  B <-->|WebSocket 鐢婚潰+杈撳叆| CA
  Phone[鎵嬫満娴忚鍣╙ -->|涓绘帶| CB
```

## 缁勪欢

| 绋嬪簭 | 瑙掕壊 | 绔彛 |
|------|------|------|
| `lan-remote-server` | 娉ㄥ唽涓績 + 闂ㄦ埛锛堝悓涓€绔彛锛屼笉鎶婅嚜宸辨敞鍐屼负鍙鎺ц澶囷級 | TCP **8765** |
| `lan-remote-client` | 姣忓彴鐢佃剳锛氬彲琚帶 + 鍙富鎺?| TCP **8765** |

- **Server** 涓嶆埅灞忋€佷笉娉ㄥ叆杈撳叆锛涚淮鎶よ澶囩洰褰曪紝骞舵彁渚涚綉椤甸棬鎴蜂笌鎺у埗浠ｇ悊銆?- **Client** 璁剧疆鏈満 PIN銆佸～ Service IP锛涙棦鍙鍒汉鎺э紝涔熷彲鎺у埗鍒汉銆?- 澶氱綉鍗′細娉ㄥ唽鍏ㄩ儴 IPv4锛涜繛鎺ユ椂鑷姩灏濊瘯锛屼笉鍚?`127.0.0.1`銆?
## 蹇€熷紑濮?
浠?**[Releases](https://github.com/hyc-yuchen/lan-remote/releases/latest)** 涓嬭浇锛?
| 鏂囦欢 | 璇存槑 |
|------|------|
| `lan-remote-server.exe` | 涓績鏈猴紙娉ㄥ唽涓績 + 闂ㄦ埛锛塛indows |
| `lan-remote-client.exe` | 鍚勭數鑴?Windows |
| `lan-remote_1.2.3_amd64.deb` | Linux Debian/Ubuntu 涓€閿畨瑁?|
| `lan-remote-*-linux` | Linux 瑁镐簩杩涘埗 |
| `open-firewall.bat` | 闃茬伀澧欐斁琛岋紙绠＄悊鍛樿繍琛岋級 |

### 1. 涓績鏈?
```bat
lan-remote-server.exe
```

- 绠＄悊椤碉細`http://涓績鏈篒P:8765/server/`
- 缁熶竴鎺у埗鍏ュ彛锛歚http://涓績鏈篒P:8765`锛堟墜鏈?娴忚鍣ㄧ敤杩欎釜锛?- Service 鍦板潃锛坈lient 濉繖涓級锛歚http://涓績鏈篒P:8765/server`

### 2. 鍚勭數鑴?
```bat
lan-remote-client.exe
```

棣栨鍚姩锛?
1. 璁剧疆鏈満 **PIN**锛堝彲闅忔満鐢熸垚锛屼細淇濆瓨锛?2. 濉?**Service IP**锛堜腑蹇冩満 IP锛屽 `192.168.1.10`锛?3. 鍚姩

涔嬪悗鍦ㄣ€屽眬鍩熺綉璁惧銆嶇偣瀵规柟 鈫?杈撳鏂?PIN 鈫?杩滅▼鎺у埗銆?
### 3. 鎵嬫満

涓庣數鑴戝悓涓€ Wi-Fi锛屾墦寮€锛?
```
http://涓績鏈篒P:8765
```

鍒楄〃閫夎澶囨垨鎵嬪姩杩炴帴 `鐩爣IP:8765` + PIN銆?
## 鍔熻兘

- 瀹炴椂灞忓箷鎺ㄦ祦锛圝PEG/WebSocket锛夛紝鐢昏川 1鈥?00銆佸抚鐜囨渶楂?120
- 榧犳爣銆侀敭鐩樸€佷腑鏂囨枃瀛楁敞鍏?- 瑙︽懜锛氱偣鎸夊乏閿€侀暱鎸夊彸閿€佸弻鎸囨粴鍔?- 杩滅▼鍏ㄥ睆
- **鏂囦欢浜掍紶**锛氬弻鏍忥紙鏈満 鈫?杩滅锛夛紝鍙祻瑙堣繙绔叏閮ㄧ鐩?鐩綍锛屼笂浼犲埌褰撳墠鐩綍銆佸嬀閫変笅杞?- 璁惧娉ㄥ唽涓庡績璺筹紙绾?20 绉掍笅绾匡級
- Windows 鎵樼洏锛氬叧绐楄繘鎵樼洏锛岃彍鍗?鍙屽嚮鍙噸鏂版墦寮€
- `-bg` 鍚庡彴妯″紡锛堟棩蹇楀啓鏂囦欢锛屾棤鎺у埗鍙帮級
- Server 绠＄悊椤碉紙:8760锛変笌缁熶竴闂ㄦ埛锛?8765锛?- 鍒嗚鲸鐜囪嚜閫傚簲涓?DPI 鎰熺煡

## 鍛戒护琛?
**server**

| 鍙傛暟 | 榛樿 | 璇存槑 |
|------|------|------|
| `-port` | 8765 | 缁熶竴绔彛锛堟敞鍐?闂ㄦ埛锛?|
| `-no-gui` | false | 涓嶅缓绐楀彛 |
| `-bg` | false | 鍚庡彴杩愯 |

**client**

| 鍙傛暟 | 榛樿 | 璇存槑 |
|------|------|------|
| `-port` | 8765 | 鎺у埗鍙?|
| `-pin` | 锛堢┖锛?| 鏈満 PIN |
| `-hub` | 锛堢┖锛?| Service `host[:port]`锛屼篃鍙湪鐣岄潰濉?|
| `-q` / `-fps` | 70 / 15 | 鐢昏川 / 甯х巼 |
| `-no-gui` / `-bg` | false | 鏃犵獥鍙?/ 鍚庡彴 |

## 閰嶇疆鏂囦欢

| 骞冲彴 | 璺緞 |
|------|------|
| Windows | `%APPDATA%\lan-remote\server.json`銆乣client.json` |
| Linux | `~/.config/lan-remote/` |

## 骞冲彴璇存槑

| 骞冲彴 | 琚帶 | 涓绘帶 |
|------|------|------|
| Windows | 鉁?| 鉁?鍘熺敓绐楀彛锛圵ebView2锛?|
| Linux X11 | 鉁?闇€ `scrot` 鎴?ImageMagick锛屼互鍙?`xdotool` | 鉁?|
| Android / iOS | 鉂?闇€鍙﹀啓鍘熺敓 App | 鉁?娴忚鍣ㄦ墦寮€ `:8765` |
| Wayland | 鉂?鍏ㄥ眬鎴睆鍙楅檺 | 鉁?|

Linux 渚濊禆锛?
```bash
sudo apt install scrot imagemagick xdotool
```

Linux 瀹夎锛堟帹鑽?Debian/Ubuntu锛夛細

```bash
sudo dpkg -i lan-remote_1.2.3_amd64.deb
# 鑻ョ己渚濊禆锛歴udo apt-get install -f
```

閫氱敤 Linux锛堣В鍘嬪悗锛夛細

```bash
sudo ./install.sh
```

Linux 渚濊禆锛堣鎺ч渶瑕侊級锛?
```bash
sudo apt install scrot imagemagick xdotool
```

## 鍚庡彴杩愯

**Windows**

- 鍏崇獥 鈫?鎵樼洏锛涙墭鐩樿彍鍗曘€屾樉绀虹獥鍙ｃ€嶆垨鍙屽嚮鍥炬爣
- `-bg`锛氭棤鎺у埗鍙帮紝鏃ュ織鍐欎复鏃剁洰褰?
**Linux**

```bash
./lan-remote-server -bg
# 鎴?nohup ./lan-remote-server -bg >/dev/null 2>&1 &
```

鏈夋闈細璇濇椂涔熷彲鎵樼洏锛涙棤妗岄潰绾悗鍙般€?
## 闃茬伀澧?
- TCP **8765**锛堢粺涓€绔彛锛氭敞鍐?/ 闂ㄦ埛 / 鎺у埗浠ｇ悊锛?
鍙敤 `open-firewall.bat`锛堢鐞嗗憳锛変竴閿斁琛?Private/Domain銆?
鑻ュ悓缃戜粛杩炰笉涓婏紝妫€鏌ヨ矾鐢卞櫒銆孉P 闅旂 / 璁垮闅旂銆嶃€?
### SmartScreen

棣栨杩愯鏈鍚?exe 鍙兘琚?Windows SmartScreen 鎷︽埅锛氱偣 **鏇村淇℃伅 鈫?浠嶈杩愯**銆傝繖鏄湭浠ｇ爜绛惧悕瀵艰嚧锛屼笉鏄▼搴忔晠闅溿€?
## 浠庢簮鐮佹瀯寤?
闇€瑕?Go 1.21+銆?
```bash
git clone https://github.com/hyc-yuchen/lan-remote.git
cd lan-remote

go build -ldflags "-s -w -H windowsgui" -o dist/lan-remote-server.exe ./cmd/server
go build -ldflags "-s -w -H windowsgui" -o dist/lan-remote-client.exe ./cmd/client

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/lan-remote-server ./cmd/server
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/lan-remote-client ./cmd/client
```

鎴?`make server client`銆?
棰勭紪璇戝寘璇峰埌 **[Releases](https://github.com/hyc-yuchen/lan-remote/releases)** 涓嬭浇銆?
閲嶆柊鐢熸垚鍥炬爣锛?
```bash
go run tools/genicon/main.go
# 鍐嶇敤 rsrc 鐢熸垚 cmd/*/rsrc.syso
```

## 鐩綍缁撴瀯

```
cmd/server/          娉ㄥ唽涓績 + 闂ㄦ埛鍏ュ彛
cmd/client/          绔晶鍏ュ彛锛堟埅灞?+ 鎺у埗 UI锛?internal/capture/    灞忓箷鎴彇锛圵in / Linux锛?internal/inject/     閿紶娉ㄥ叆
internal/registry/   娉ㄥ唽涓績涓庡鎴风銆佺鐞嗛〉
internal/portal/     缁熶竴闂ㄦ埛涓?WS/鏂囦欢浠ｇ悊
internal/server/     HTTP + WebSocket 鎺у埗鍗忚 + 鍐呭祵缃戦〉
internal/config/     閰嶇疆璇诲啓
internal/appwin/     绐楀彛 / 鎵樼洏 / 鍥炬爣
internal/tray/       绯荤粺鎵樼洏
web/                 缃戦〉婧愮爜锛堟瀯寤烘椂 embed锛?tools/genicon/       鐢熸垚 ICO
```

## 鍗忚绠€杩?
1. Client 鍚?`POST /api/register` 鎴?`/api/heartbeat` 涓婃姤鍚嶇О涓庡叏閮?IP銆?2. 鍏朵粬绔?`GET /api/devices` 鎷夊彇鍦ㄧ嚎鍒楄〃銆?3. 涓绘帶杩?`ws://鐩爣:8765/ws`锛屽厛鍙?`{"type":"auth","pin":"..."}`銆?4. 閴存潈閫氳繃鍚庢湇鍔＄鎺?JPEG 浜岃繘鍒跺抚锛涘鎴风鍙?`move` / `button` / `key` / `text` / `scroll`銆?5. 鏂囦欢锛歚POST /api/file` 涓婁紶锛宍GET /api/files` 鍒楃洰褰曪紝`GET /api/download` 涓嬭浇锛堥渶 PIN锛夈€?
## 瀹夊叏璇存槑

- PIN 鐢ㄤ簬鍚岀綉璁惧浜掕锛涜鍕挎妸绔彛鐩存帴鏆撮湶鍒板叕缃戙€?- 淇敼鏈満 PIN 浠呭厑璁稿洖鐜闂€?- 杩滅鏂囦欢娴忚鏃犵洰褰曟矙绠憋紝PIN 閫氳繃鍗冲彲璁块棶鏈満浠绘剰璺緞鈥斺€旇浠呭湪鍙椾俊灞€鍩熺綉浣跨敤銆?- 鏈娇鐢?TLS锛涜法涓嶅彲淇＄綉缁滆鑷鍔?VPN/闅ч亾銆?
## License

MIT
