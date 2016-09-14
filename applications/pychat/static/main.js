var config;

function lambda_post(data, callback) {
  var url = config['url'];
  // replace host with window.location.hostname
  var host = /(.*:\/\/|^)([^\/]+)/.exec(url)[2];
  var port = host.split(':')[1];
  port = port == undefined ? "" : ":" + port;
  url = url.replace(host, window.location.hostname+port);

  $.ajax({
    type: "POST",
    url: url,
    contentType: "application/json; charset=utf-8",
    data: JSON.stringify(data),
    dataType: "json",
    success: callback,
    failure: function(error) {
      $("#comments").html("Error: " + error + ".  Consider refreshing.")
    }
  });  
}

function comment() {
  var msg = $("#comment").val();
  lambda_post({"op":"msg", "msg":msg}, function(data){
    $("#comment").val("");
  });
}

function updates(ts) {
  var data = {"op":"updates", "ts":ts};
  lambda_post(data, function(data){
    if ("error" in data) {
      html_error = data.error.replace(/\n/g, '<br/>');
      $("#comments").html("Error: <br/><br/>" + html_error + "<br/><br/>Consider refreshing.")
    } else {
      var ts = 0;
      for (var i = 0, len = data.result.length; i < len; i++) {
	row = data.result[i];
	$("#comments").append(row.msg + "<br/>");
	if (row.ts > ts) {
	  ts = row.ts;
	}
      }
      updates(ts);
    }
  });
}

function main() {
  $("#comments").html("initializing");
  $.getJSON('config.json')
    .done(function(data) {
      config = data;
      $("#comments").html("");

      // setup handlers
      $('#comment').keypress(function(e){
	if(e.keyCode==13)
	  $('#submit').click();
      });
      $("#submit").click(comment);
      updates(0);
    })
    .fail(function( jqxhr, textStatus, error ) {
      $("#comments").html("Error: " + error + ".  Consider refreshing.")
    })
}

$(document).ready(function() {
  main();
});
