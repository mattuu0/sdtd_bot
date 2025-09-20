$env:GOOS="linux"; $env:GOARCH="amd64"; go build -ldflags="-s -w"  -o sdtdbot .
scp ./sdtdbot root@7days:/home/sdtdserver/sdtdbot
scp ./.env root@7days:/home/sdtdserver/.env
ssh root@7days -t chmod +x /home/sdtdserver/sdtdbot
