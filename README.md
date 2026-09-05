# LAN Remote

灞€鍩熺綉杩滅▼妗岄潰鎺у埗銆俉indows / Linux 浜掓帶锛屾墜鏈烘祻瑙堝櫒鍙綔涓绘帶绔€?
**涓嬭浇瀹夎鍖?鈫?[Releases](https://github.com/hyc-yuchen/lan-remote/releases/latest)**

```mermaid
graph LR
  subgraph S [lan-remote-server 娉ㄥ唽涓績]
    R[娉ㄥ唽鍙?:8760]
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
| `lan-remote-server` | 娉ㄥ唽涓績 / Service锛屽彧璐熻矗璁惧鐩綍 | TCP **8760** |
| `lan-remote-client` | 姣忓彴鐢佃剳涓婄殑绔細鍙鎺?+ 鍙富鎺?| TCP **8765** |

- **Server** 涓嶆埅灞忋€佷笉娉ㄥ叆杈撳叆锛屽彧缁存姢鍦ㄧ嚎璁惧鍒楄〃銆?- **Client** 鍚姩鍚庤缃湰鏈?PIN銆佸～ Service IP锛涙棦鍙鍒汉鎺э紝涔熷彲鍦ㄧ晫闈㈤噷鎺у埆浜恒€?- 澶氱綉鍗℃満鍣ㄤ細娉ㄥ唽**鍏ㄩ儴 IPv4**锛岃繛鎺ユ椂鑷姩閫愪釜灏濊瘯銆?
## 蹇€熷紑濮?
浠?**[Releases](https://github.com/hyc-yuchen/lan-remote/releases/latest)** 涓嬭浇锛?
| 鏂囦欢 | 璇存槑 |
|------|------|
| `lan-remote-server.exe` | 涓績鏈猴紙娉ㄥ唽涓績锛塛indows |
| `lan-remote-client.exe` | 鍚勭數鑴?Windows |
| `lan-remote-server-linux` / `lan-remote-client-linux` | Linux |
| `open-firewall.bat` | 闃茬伀澧欐斁琛岋紙绠＄悊鍛樿繍琛岋級 |
| `*-windows.zip` / `*-linux.zip` | 鎵撳寘涓嬭浇 |

### 1. 鍚姩娉ㄥ唽涓績锛堜竴鍙板父寮€鐨勭數鑴戯級

```bat
lan-remote-server.exe
```

璁颁笅灞忓箷涓婄殑 Address锛屼緥濡?`http://192.168.1.10:8760`銆?
### 2. 姣忓彴鍙備笌鐨勭數鑴戝惎鍔?Client

```bat
lan-remote-client.exe
```

棣栨鍚姩鍦ㄧ獥鍙ｉ噷锛?
1. 璁剧疆鏈満 **PIN**锛堝彲銆岄殢鏈虹敓鎴愩€嶏級
2. 濉?**Service IP**锛堟敞鍐屼腑蹇?IP锛屽 `192.168.1.10`锛?3. 鐐广€屽惎鍔ㄣ€?
涔嬪悗鍦ㄣ€屽眬鍩熺綉璁惧銆嶉噷鐐瑰鏂?鈫?杈撳叆瀵规柟 PIN 鈫?杩滅▼鎺у埗銆?
### 3. 鎵嬫満涓绘帶

鎵嬫満涓庣數鑴戝悓涓€ Wi鈥慒i锛屾祻瑙堝櫒鎵撳紑鏌愬彴 Client锛?
```
http://鐢佃剳IP:8765
```

杩滅▼璁块棶浼氳繘鍏ユ帶鍒剁妯″紡锛屽湪鍒楄〃涓€夎澶囷紝鎴栥€屾墜鍔ㄨ繛鎺ャ€嶅～ `鐩爣IP:8765` + PIN銆?
## 鍛戒护琛屽弬鏁?
**server**

| 鍙傛暟 | 榛樿 | 璇存槑 |
|------|------|------|
| `-port` | 8760 | 娉ㄥ唽鍙?|

**client**

| 鍙傛暟 | 榛樿 | 璇存槑 |
|------|------|------|
| `-port` | 8765 | 鎺у埗鍙?|
| `-pin` | 锛堢┖锛?| 鏈満 PIN锛屼細鍐欏叆閰嶇疆 |
| `-hub` | 锛堢┖锛?| Service `host[:port]`锛屼篃鍙湪鐣岄潰濉?|
| `-q` / `-fps` | 70 / 15 | 鐢昏川涓庡抚鐜?|
| `-no-gui` | false | 涓嶅缓绐楀彛锛岀敤娴忚鍣ㄦ墦寮€ UI |

## 閰嶇疆鏂囦欢

| 骞冲彴 | 璺緞 |
|------|------|
| Windows | `%APPDATA%\lan-remote\server.json` / `client.json` |
| Linux | `~/.config/lan-remote/` |

PIN 涓?Service 鍦板潃淇濆瓨鍦ㄩ厤缃噷锛屼笅娆″惎鍔ㄨ嚜鍔ㄥ姞杞姐€?
## 鍔熻兘

- 瀹炴椂灞忓箷鎺ㄦ祦锛圝PEG / WebSocket锛屽彲璋冪敾璐ㄤ笌 FPS锛?- 榧犳爣绉诲姩 / 宸﹀彸閿?/ 婊氳疆锛岄敭鐩樹笌涓枃鏂囧瓧娉ㄥ叆
- 鎵嬫満瑙︽帶锛氱偣鎸夊乏閿€侀暱鎸夊彸閿€佸弻鎸囨粴鍔?- 璁惧鑷姩娉ㄥ唽涓庡績璺筹紙绾?20s 涓嬬嚎锛?- Windows 鍘熺敓绐楀彛锛圵ebView2锛夛紝鏃犻粦妗嗘帶鍒跺彴
- 鍒嗚鲸鐜囪嚜閫傚簲锛圖PI 鎰熺煡鎴睆 + 鐢诲竷绛夋瘮缂╂斁锛?
## 骞冲彴璇存槑

| 骞冲彴 | 琚帶 | 涓绘帶 |
|------|------|------|
| Windows | 鉁咃紙Win32 BitBlt + SendInput锛?| 鉁?|
| Linux X11 | 鉁咃紙闇€ `scrot` 鎴?ImageMagick锛屼互鍙?`xdotool`锛?| 鉁?|
| Android / iOS | 鉂岋紙闇€鍙﹀啓鍘熺敓 App锛?| 鉁?娴忚鍣ㄨ闂?`:8765` |
| Wayland | 鉂?鍏ㄥ眬鎴睆鍙楅檺 | 鉁?|

### Linux 渚濊禆

```bash
sudo apt install scrot imagemagick xdotool
```

### Windows WebView2

Client 绐楀彛渚濊禆 [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/)锛圵in10/11 涓€鑸嚜甯︼級銆?
## 闃茬伀澧?
鏀捐锛?
- TCP **8765**锛堟帶鍒讹級
- TCP **8760**锛堟敞鍐岋級

浠撳簱鍐?`open-firewall.bat` 鍙?*浠ョ鐞嗗憳韬唤杩愯**涓€閿斁琛岋紙Private/Domain锛夈€?
鍚岀綉鑻ヤ粛杩炰笉涓婏紝妫€鏌ヨ矾鐢卞櫒鏄惁寮€鍚€孉P 闅旂 / 璁垮闅旂銆嶃€?
## 浠庢簮鐮佹瀯寤?
闇€瑕?[Go](https://go.dev/dl/) 1.21+銆?
```bash
git clone https://github.com/hyc-yuchen/lan-remote.git
cd lan-remote

# Windows
go build -ldflags "-s -w -H windowsgui" -o dist/lan-remote-server.exe ./cmd/server
go build -ldflags "-s -w -H windowsgui" -o dist/lan-remote-client.exe ./cmd/client

# Linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/lan-remote-server ./cmd/server
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/lan-remote-client ./cmd/client
```

鎴栵細

```bash
make server client
```

棰勭紪璇戝寘璇峰埌 **[Releases](https://github.com/hyc-yuchen/lan-remote/releases)** 涓嬭浇锛屼笉瑕佷粠婧愮爜鏍戦噷鎵惧畨瑁呭寘銆?
## 瀹夊叏璇存槑

- PIN 浠呯敤浜庡悓缃戣澶囦簰璁わ紝**涓嶈**鎶婃帶鍒跺彛鐩存帴鏆撮湶鍒板叕缃戙€?- 淇敼 PIN 鐨勬帴鍙ｄ粎鍏佽鏈満鍥炵幆璁块棶銆?- 鏈娇鐢?TLS锛涜法涓嶅彲淇＄綉缁滆鑷鍔?VPN / 闅ч亾銆?
## 鐩綍缁撴瀯

```
cmd/server/          娉ㄥ唽涓績鍏ュ彛
cmd/client/          绔晶鍏ュ彛锛堟埅灞?+ 鎺у埗 UI锛?internal/capture/    灞忓箷鎴彇锛圵in / Linux锛?internal/inject/     閿紶娉ㄥ叆
internal/registry/   娉ㄥ唽涓績鏈嶅姟涓庡鎴风
internal/server/     HTTP + WebSocket 鎺у埗鍗忚 + 鍐呭祵缃戦〉
internal/config/     閰嶇疆璇诲啓
internal/appwin/     WebView2 / 娴忚鍣ㄧ獥鍙?web/                 缃戦〉婧愮爜锛堟瀯寤烘椂 embed锛?```

## 鍗忚绠€杩?
1. Client 鍚?`POST /api/register`锛堟垨蹇冭烦 `/api/heartbeat`锛変笂鎶ュ悕绉颁笌鍏ㄩ儴 IP銆?2. 鍏朵粬绔?`GET /api/devices` 鎷夊彇鍦ㄧ嚎鍒楄〃銆?3. 涓绘帶杩?`ws://鐩爣:8765/ws`锛屽厛鍙?`{"type":"auth","pin":"..."}`銆?4. 閴存潈閫氳繃鍚庢湇鍔＄鎺?JPEG 浜岃繘鍒跺抚锛涘鎴风鍙?`move` / `button` / `key` / `text` / `scroll` JSON銆?
## License

MIT

