var config;
var requestNum;
var currPref = '';
function lambda_post(data, callback) {
  var url = config['url'];
  $.ajax({
    type: "POST",
    url: url,
    contentType: "application/json; charset=utf-8",
    data: JSON.stringify(data),
    dataType: "json",
    timeout: 1000,
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
    else if (e.keyCode >= 65 && e.keyCode < 90){
        $("#text").keyup(function() {
            keystroke()});
    }
    else {
        clearsuggs();
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

function log(last, curr) {
    console.log(last, requestNum, curr);
}
    
function keystroke() {
    var entries = $("#text").val();
    var words  = entries.split(' ');
    var lastword = words[words.length - 1];
    if (lastword.includes("'")) {
        clearsuggs();
        return;
    }
    if (lastword.localeCompare(currPref) != 0) {
        var currReq;
        currReq = ++requestNum;
        //console.log("=====================")
        //console.log("PRE REQUEST:", lastword);
        //log(lastword, requestNum, currReq);
        currPref = lastword;
        //console.log("SEND?:", lastword);
        //log(lastword, requestNum, currReq);
        if (currReq == requestNum) {
            //console.log("SENT");
            lambda_post({"op": "keystroke", "pref": lastword}, function(data){
                try {
                    //console.log("DISPLAY RESULTS?:", lastword)
                    //log(lastword, requestNum, currReq)
                    if (currReq == requestNum) {
                        updatesuggs(data.result);
                      //  console.log("YES");
                    }
                    else {
                        //console.log("IGNORED");
                        //clearsuggs();
                    }

                }
                catch(err) {
                    clearsuggs();
                }
            });
        }
        else {
            //console.log("NOT SENT")
        }
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
    requestNum = 0;

  $("#initmsg").html("ready");
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
