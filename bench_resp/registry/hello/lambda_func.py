import time
def handler(conn, event):
    try:
        return "Hello hello , greenT!"
    except Exception as e:
        return {'error': str(e)} 
