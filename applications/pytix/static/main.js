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
      $("#seatmap").html("Error: " + error + ".  Consider refreshing.")
    }
  });  
}

function clear() {
  lambda_post({"op":"init"}, function(data){
    // pass
  });
}

function hold(snum) {
  var data = {"op":"hold", "snum":snum};
  lambda_post(data, function(data){
    alert(data);
    // pass
  });
}

function book() {
  lambda_post({"op":"book"}, function(data){
    // pass
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
        if (data.result.smap[i] = 'free') {
          $("#snum_" + i).append(
            '</td><td><button snum=' + i + 
            ' class="hold">Hold</button></td></tr>'
          );
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

      // setup handlers
      $("#clear").click(clear);
      $("#book").click(book);
      $(".hold").on("click", function(){
        hold($(this).attr("snum"));
      });

      updates(0);
    })
    .fail(function( jqxhr, textStatus, error ) {
      $("#seatmap").html("Error: " + error + ".  Consider refreshing.")
    })
}

$(document).ready(function() {
  main();
});
