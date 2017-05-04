import socket 

def handler(conn, event):
    try:
      port = int(event['port'])

      sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
      #sock.bind("172.17.0.1", port)
      sock.connect(("0.0.0.0", port))
      return "Call: done"
    except Exception as e:
        return {'error': str(e)} 
