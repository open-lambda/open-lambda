import socket 

sock_addr = ("localhost", 4567)

def handler(conn, event):
    try:
      sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
      sock.bind(sock_addr)

      sock.listen(1)
      connection, client_addr = sock.accept()
      return "Resp: done"
    except Exception as e:
        return {'error': str(e)} 
