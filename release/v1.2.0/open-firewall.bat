@echo off
REM Run as Administrator: right-click -> Run as administrator
netsh advfirewall firewall add rule name="LAN Remote Control 8765" dir=in action=allow protocol=TCP localport=8765 profile=private,domain
netsh advfirewall firewall add rule name="LAN Remote Registry 8760" dir=in action=allow protocol=TCP localport=8760 profile=private,domain
echo.
echo Done. Ports 8765 (control) and 8760 (registry) are allowed on Private/Domain networks.
echo Make sure phone Wi-Fi is on the SAME network as this PC.
pause
