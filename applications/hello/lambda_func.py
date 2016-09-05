def handler(conn, event):
    try:
        return "Hello, %s!" % event['name']
    except Exception as e:
        return {'error': str(e)} 
