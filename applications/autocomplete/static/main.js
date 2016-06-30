var config;
var prevReq;
var currPref = '';
function lambda_post(data, callback) {
  var url = config['url'];
  prevReq = $.ajax({
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
function checkenter(e) {
    if (e.keyCode == 13 && !e.shiftKey || e.keycode == 49 && e.ctrlKey )  {
        e.preventDefault();
        completeword(1);
        clearsuggs();
    }
    else if (e.ctrlKey) {
        if (e.keyCode == 50) {
            completeword(2);
        }
        else if (e.keyCode == 51) {
            completeword(3);
        }
        else if (e.keyCode == 52) {
            completeword(4);
        }
        else if (e.keyCode == 53) {
            completeword(5);
        }
    }
    else if (e.charCode == 32) {
        clearsuggs();
    }
    else {
        $("#text").keyup(function() {
            keystroke()});
    }
}
function completeword(s) {
    var currsugg = '#sugg' + s;
    var compword = $(currsugg).html();
    var entries = $('#text').val();
    var replaced = entries.replace(/\w*$/, compword);
    $('#text').val(replaced);
    clearsuggs();
    $('#text').focus();
}

function keystroke() {
    var entries = $("#text").val();
    var words  = entries.split(' ');
    var lastword = words[words.length - 1];
    if (lastword.contains("'")) {
        clearsuggs();
        return;
    }
    if (lastword.localeCompare(currPref) != 0) {
        currPref = lastword;

        try{
            prevReq.abort();
        }
        catch(err){
        };

        lambda_post({"op": "keystroke", "pref": lastword}, function(data){
            try {
                prevReq = null;
                updatesuggs(data.result);
            }
            catch(err) {
                clearsuggs();
            }
        });
    }
}
function clearsuggs() {
    $('#sugg1').html('-');
    $('#sugg2').html('-');
    $('#sugg3').html('-');
    $('#sugg4').html('-');
    $('#sugg5').html('-');
}
function updatesuggs(suggs) {
    var currsugg;
    var i;
    var j;
    for (i = 1; i < suggs.length + 1; i++) {
        currsugg = "#sugg" + i;
        $(currsugg).html(suggs[i-1]);
    }
    for (j = i; j < 6; j++) {
        currsugg = "#sugg" + j;
        $(currsugg).html('');
    }
}
function main() {

  $("#initmsg").html("ready");
  //init();
  $.getJSON('config.json')
    .done(function(data) {
      config = data;
      $('#text').keydown(function(e) {
          checkenter(e)});
    })

    .fail(function( jqxhr, textStatus, error ) {
      $("#initmsg").html("Error: " + error + ".  Consider refreshing.")
    })
}
$(document).ready(function() {
  main();
});
