var config;
var holdButton = '</td><td><button class="hold">Hold</button></td></tr>';

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
      $("#seatmap").html("Error: " + error + ".  Consider refreshing.")
    }
  });  
}

function clear() {
  lambda_post({"op":"init"}, function(data){
    // pass
  });
}

function hold() {
  lambda_post({"op":"hold"}, function(data){

  });
}

function updates(ts) {
  var data = {"op":"updates", "ts":ts};
  lambda_post(data, function(data){
    if ("error" in data) {
      html_error = data.error.replace(/\n/g, '<br/>');
      $("#seatmap").html("Error: <br/><br/>" + html_error + "<br/><br/>Consider refreshing.")
    } else {
      if ($("#snum_" + data.result.snum).length == 0) {
        addSeat(data.result.snum, data.result.stat);
      } else {
        // TODO replace status in row
      }
      updates(data.result.ts);
    }
  });
}

function addSeat(snum, stat) {
  $("#seatmap").append('<tr id=snum_' + snum + '><td>' + snum + '</td><td>' + stat + holdButton);
}

function main() {
  $.getJSON('config.json')
    .done(function(data) {
      config = data;

      // setup handlers
      $("#clear").click(clear);

      updates(0);
    })
    .fail(function( jqxhr, textStatus, error ) {
      $("#seatmap").html("Error: " + error + ".  Consider refreshing.")
    })
}

$(document).ready(function() {
  main();
});
