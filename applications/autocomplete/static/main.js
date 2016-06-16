//var config;

function lambda_post(data, callback) {
    
  var url = config['url']
  //var url = "http://107.170.82.34:32779/runLambda/sueoshwciyig";
  if (data['op'] == 'keystroke')
      $('#initmsg').html('fuck');



  $.ajax({
    type: "POST",
    url: url,
    contentType: "application/json; charset=utf-8",
    data: JSON.stringify(data),
    dataType: "json",
    success: callback,
    failure: function(error) {
      $("#initmsg").html("Error: " + error + ".  Consider refreshing.")
    }
  });  
}

function init() {
   
    lambda_post({"op": "init"}, function(data){
        $("#initmsg").html('ready');
    });
}

function keystroke() {
    //var entries = $("#text").text().split(" ");
    //var currword = entries[entries.length - 1];
    //$('#sugg1').html('hi');
    lambda_post({"op": "keystroke", "pref": 'x'}, function(data){
        //updatesuggs(data.result.result);
        $('#sugg1').html('hi');
        $('#sugg2').html(data.result);
        $('#sugg3').html(data.result[2]);
        $('#sugg4').html(data.result[3]);
        $('#sugg5').html(data.result[4]);
    });

}
function updatesuggs(suggs) {
    if (suggs == 'clear'){
        $('#sugg1').html('');
        $('#sugg2').html('');
        $('#sugg3').html('');
        $('#sugg4').html('');
        $('#sugg5').text('');
    }
    else {
        $('#sugg1').html(suggs[0]);
        $('#sugg2').html(suggs[1]);
        $('#sugg3').html(suggs[2]);
        $('#sugg4').html(suggs[3]);
        $('#sugg5').html(suggs[4]);

    }
}
function main() {

  $("#initmsg").html("initializin");
  //init();
  $.getJSON('config.json')
    .done(function(data) {
      // setup handlers
      $('#text').keypress(function(e){
          if(e.keyCode == 13 || e.keyCode == 32){
              updatesuggs('clear');
              //keystroke();
          }
          else if (e.keyCode != 32){
              keystroke();
          }
      });
    })

    .fail(function( jqxhr, textStatus, error ) {
      $("#initmsg").html("Error: " + error + ".  Consider refreshing.")
    })
}

$(document).ready(function() {
  main();
});
