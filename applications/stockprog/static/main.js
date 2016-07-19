var STOCK_PERIOD = 5000; //Period for stock get (defined in ms).
var config;
var lastStock = "YHOO"; //default
var timerCall = 0;
function lambda_post(data, callback) {
  var url = config['url']

  $.ajax({
    type: "POST",
    url: url,
    contentType: "application/json; charset=utf-8",
    data: JSON.stringify(data),
    dataType: "json",
    success: callback,
    failure: function(error) {
      $("#old_stocks").html("Error: " + error + ".  Consider refreshing.")
    }
  });  
}

function clear() {
  lambda_post({"op":"init"}, function(data) {
  });
}

//Request stock quote from backend.
function get_quote() {

  //Call the get_stock method in py backend.
  var stock_id = $("#stock_name").val();

  //If no id entered, then use default (for timed requests).
  if (stock_id == "")
    stock_id = lastStock;

  var cmd = {"op":"get_stock", "stock_id":stock_id};
 
  lambda_post(cmd, function(data) {
    //Make sure the lastStock is always retrievable.
    if (data.result.status != "error")
       lastStock = stock_id;

    //Make sure calls by timer don't clear the textfield.
    if (timerCall != 1)
      $("#stock_name").val("");
    timerCall = 0;

    //updates(data.result.ts);
    //alert(lastStock);
  });
}


//Update the HTML output window according to new stock data in the DB.
function updates(ts) {

  //Request the latest data (uses timestamp to find newest).
  var data = {"op":"updates", "ts":ts};
  lambda_post(data, function(data){

    if ("error" in data) {

      html_error = data.error.replace(/\n/g, '<br/>');
      $("#old_stocks").html("Error: <br/><br/>" + html_error
        + "<br/><br/>Consider refreshing.")
    } else {

      //Access parts of the retrieved row using column names.
      $("#old_stocks").append("Stock: " + data.result.old_stock +
        "  PRICE: " + data.result.stock_price + "<br/>");
      //Update again.
      updates(data.result.ts);
    }
  });
}



function main() {

  //Initialize DB url.
  $("#old_stocks").html("Initializing");
  $.getJSON('config.json')
    .done(function(data) {
      config = data;
      $("#old_stocks").html("");


      //Bind events to HTML ids.
      $("#stock_name").keypress(function(e) {
        //Enter in textfield simulates clicking check_stock.
        if (e.keyCode==13)
          $("#check_stock").click();
      });
      $("#check_stock").click(get_quote);
      $("#clear").click(clear);

      //Begin periodic stock value requests.
      get_quote_periodic();
      //Begin repeated updates.
      updates(0);
    })
    .fail(function( jqxhr, textStatus, error ) {
      $("#old_stocks").html("Error: " + error + ".  Consider refreshing.")
    })
}


function get_quote_periodic() {

  timerCall = 1;
  //display alert for every 5s + works for on-click.
  get_quote();

  //Call timer_ticks() again after 5s.
  setTimeout(get_quote_periodic, STOCK_PERIOD);

}

 
//Pass to the document element's ready function, the function main();(?)
$(document).ready(function() {
  main();
});
