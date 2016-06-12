var config;
var unum;

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
    return data.result;
  });
}

function hold(snum) {
  var data = {"op":"hold", "snum":snum, "unum":unum};
  lambda_post(data, function(data){
    if (data.result['replaced'] != 1) {
      $("#alert_" + snum).html(" Already held!");
    }
    return data.result;
  });
}

function book() {
  lambda_post({"op":"book", "unum":unum}, function(data){
    return data.result;
  });
}

// TODO how to determine if this is really the newest? how to manage?
function updates(ts) {
  var data = {"op":"updates", "ts":ts};
  lambda_post(data, function(data){
    if ("error" in data) {
      html_error = data.error.replace(/\n/g, '<br/>');
      $("#seatmap").html("Error: <br/><br/>" + html_error + "<br/><br/>Consider refreshing.")
    } else {
      $("#seatmap").html('<tr><td>Number</td><td>Status</td><td>Action</td></tr>');
      for (var i = 1; i <= data.result['max']; i++) {
        $("#seatmap").append(
          '<tr id=snum_' + i +
          '><td>'        + i +
          '</td><td>'    + data.result.smap[i]
        );
        if (data.result.smap[i] == 'free') {
          $("#snum_" + i).append(
            '</td><td><button snum='           + i + 
            ' class="hold" id=btn_'            + i +
            '>Hold</button></td><td id=alert_' + i +
            '></td></tr>'
          );
        } else if (data.result.umap[i] == unum) {
          $("#snum_" + i).append('<td>by you</td></tr>');
        }
      }
      updates(data.result.ts);
    }
  });
}

function main() {
  $.getJSON('config.json')
    .done(function(data) {
      config = data;

      unum = Math.floor(Math.random() * 999999999);

      updates(0);

      // setup handlers
      $("#clear").click(clear);
      $("#book").click(book);
      $("body").on("click", ".hold", function(){
        hold($(this).attr("snum"));
      });
    })
    .fail(function( jqxhr, textStatus, error ) {
      $("#seatmap").html("Error: " + error + ".  Consider refreshing.")
    })
}

$(document).ready(function() {
  main();
});
