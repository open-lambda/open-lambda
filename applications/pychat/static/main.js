var config;

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
      $("#comments").html("Error: " + error + ".  Consider refreshing.")
    }
  });  
}

function clear() {
  lambda_post({"op":"init"}, function(data){
    // pass
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
      //$("#comments").html(data.result.msg);
      $("#comments").append(data.result.msg + "<br/>");
      updates(data.result.ts);
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
      $("#clear").click(clear);
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
