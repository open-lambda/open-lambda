import time, traceback, json
from googlefinance import getQuotes
import rethinkdb as r

STOCKS = 'stocks'         # DB
STOCK_DATA = 'old_stocks' # TABLE
OLD_STOCK  = 'old_stock'  # COLUMN
STOCK_PRICE = 'stock_price' #COLUMN
TS   = 'ts'               # COLUMN


#Initialize the DB.
def init(conn, event):
 
    rv = '' #Default return value

    #Try to drop table/reset DB.
    try:
        r.db_drop(STOCKS).run(conn)
        rv = 'dropped, then created'
    except:
        rv = 'created'

    #Recreate table.
    r.db_create(STOCKS).run(conn)
    r.db(STOCKS).table_create(STOCK_DATA).run(conn)
    r.db(STOCKS).table(STOCK_DATA).index_create(OLD_STOCK).run(conn)
    r.db(STOCKS).table(STOCK_DATA).index_create(STOCK_PRICE).run(conn)
    r.db(STOCKS).table(STOCK_DATA).index_create(TS).run(conn)

    return {"status":rv}


#Get the stock quote and update the DB.
def get_stock(conn, event):
    stock_id = event["stock_id"].upper()

    try:
      stock_data = getQuotes(stock_id)
    except Exception:
      return {"status":"error"}

    stock_price = float(stock_data[0]['LastTradePrice'])
    
    #Get time of quote retrieval.
    ts = time.time()

    #Store stock data.
    r.db(STOCKS).table(STOCK_DATA).insert({OLD_STOCK:stock_id,
      STOCK_PRICE:stock_price, TS:ts}).run(conn)

    #The status of updating the DB + the timestamp for the latest stock.
    return {"status":"Stock quote retrieved", "ts":ts}


#Return the data that was updated in the DB (to push to the HTML output).
def updates(conn, event):
    ts = event.get('ts', 0) #default 0.

    #Get most recent data.
    for row in (r.db(STOCKS).table(STOCK_DATA).filter(r.row[TS] > ts).
                changes(include_initial=True).run(conn)):
        return row['new_val'] #Get the most recent row.
    #r.db(STOCKS).table(STOCK_DATA)

#Determine what the back end should do based off of the lambda_post request.
def handler(conn, event):
    #Determine function to run.
    fn = {'init':init,
          'get_stock':get_stock,
          'updates': updates}.get(event["op"], None)

    if fn != None:
        try:
            result = fn(conn, event)
            return {'result': result}
        except Exception:
            return {'error': traceback.format_exc()}
    else:
        return {'error': 'bad op'}
