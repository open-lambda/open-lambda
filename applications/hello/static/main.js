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

function main() {
  $.getJSON('config.json')
    .done(function(data) {
      config = data;
      $("#hello").html("");
      lambda_post({"op":"hello"}, function(data){
	$("#hello").html(data.result);
      });
    })
    .fail(function( jqxhr, textStatus, error ) {
      $("#comments").html("Error: " + error + ".  Consider refreshing.")
    })
}

$(document).ready(function() {
  main();
});
