var config;
var files = [];

function lambda_post(data, callback) {
  var url = config['url']

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
  ret = JSON.parse(json);
  downloadFile(ret['url'], ret['filename']);
  //var blob = base64toBlob(ret['data'], ret['datatype'])
  //var file = blobToFile(blob, ret['filename'])
}

function base64toBlob(base64Data, contentType) {
  contentType = contentType || '';
  var sliceSize = 1024;
  var byteCharacters = atob(base64Data);
  var bytesLength = byteCharacters.length;
  var slicesCount = Math.ceil(bytesLength / sliceSize);
  var byteArrays = new Array(slicesCount);

  for (var sliceIndex = 0; sliceIndex < slicesCount; ++sliceIndex) {
    var begin = sliceIndex * sliceSize;
    var end = Math.min(begin + sliceSize, bytesLength);

    var bytes = new Array(end - begin);
    for (var offset = begin, i = 0 ; offset < end; ++i, ++offset) {
      bytes[i] = byteCharacters[offset].charCodeAt(0);
    }
    byteArrays[sliceIndex] = new Uint8Array(bytes);
  }
  return new Blob(byteArrays, { type: contentType });
}

function blobToFile(blob, filename) {
    //A Blob() is almost a File() - it's just missing the two properties below which we will add
    blob.lastModifiedDate = new Date();
    blob.name = filename;
    return blob;
}

function downloadFile(URL, filename) {
  var a = document.createElement("a");
  a.download = filename;
  a.href = URL;
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
      reader.readAsDataURL(file);
    });
  });

  $("#file-form").submit(function(form) {
    $.each(files, function(index, file) {
      cmd = {'op':'ocr', 'data':file.data, 'filename':file.filename};
      lambda_post(cmd, success);
    });
    files = [];
    form.preventDefault();
  });
});