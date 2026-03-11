# 终端 1：起一个只收 POST、打印 body 并返回 200 的服务器
python3 -c '
from http.server import HTTPServer, BaseHTTPRequestHandler
import json

class H(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length)
        print("Path:", self.path)
        print("Headers:", dict(self.headers))
        try:
            print("Body (JSON):", json.dumps(json.loads(body), indent=2))
        except Exception:
            print("Body (raw):", body.decode())
        self.send_response(200)
        self.end_headers()

HTTPServer(("0.0.0.0", 9999), H).serve_forever()
'