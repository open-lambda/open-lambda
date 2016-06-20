var config;
var files = [];
var requests = {};

function lambda_post(data, callback) {
  var url = config['url']
  requests[data['filename'].split('.')[0]] = new Date().getTime();

  $.ajax({
    type: "POST",
    url: url,
    contentType: 'application/json',
    data: JSON.stringify(data),
    success: callback,
    failure: function(error) {
      alert("Error: " + error + ".  Consider refreshing.")
    }
  });
}

function success(json) {
  var ret = JSON.parse(json);
  var name = ret['filename'].split('.')[0];
  var time = new Date().getTime();
  var elapsed =  time - requests[name];
  delete requests[name];

  console.log('request for ' + name + ' took ' + (elapsed/1000) + ' s total')
  console.log('ocr time for ' + name + ' was ' + ret['ocr_time'] + ' s')
  console.log('conversion time for ' + name + ' was ' + ret['convert_time'] + ' s')

  var txt = atob(ret['data']);
  var blob = new Blob([txt], {type:"text/plain;charset=utf-8"});
  download(blob, ret['filename']);
}

function download(blob, filename) {
  var a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
}

$(document).ready(function() {
  $.getJSON('config.json')
    .done(function(data) {
      config = data;
    })
    .fail(function( jqxhr, textStatus, error ) {
      $("#comments").html("Error: " + error + ".  Consider refreshing.")
    })

  $("#files").change(function(event) {
    $.each(event.target.files, function(index, file) {
      var reader = new FileReader();
      reader.onload = function(event){
        object = {};
        object.filename = file.name;
        object.data = event.target.result;
        files.push(object);
      };
      reader.onerror = function(event) {
        alert("Failed to read file. Please try again.");
      }
      reader.readAsBinaryString(file);
    });
  });

  $("#file-form").submit(function(form) {
    $.each(files, function(index, file) {
      cmd = {'op':'ocr', 'data':file.data, 'filename':file.filename};
      lambda_post(cmd, success);
    });
    files = [];
    form.preventDefault();
    return false;
  });
});
