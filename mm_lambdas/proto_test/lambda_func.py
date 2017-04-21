import test_pb2 as p

def handler(conn, event):
    try:
      d = p.MyDict()
      d.turn = "pow"
      d.pow = "scram"
      obj = d.SerializeToString()

      e = p.MyDict()
      e.ParseFromString(obj)
      return "Recoverd obj. Turn: " + e.turn
    except Exception as e:
        return {'error': str(e)} 
