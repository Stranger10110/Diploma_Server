function Login(){
    // document.realLogin.username.value = document.fakeLogin.username.value;
    // document.realLogin.password.value = document.fakeLogin.password.value;
    // const data = new FormData(document.querySelector('#realLogin'));
    // const json_obj = Object.fromEntries(data.entries());

    $.ajax({
        type: 	'POST',
        url: 	'/api/public/login',
        data: 	JSON.stringify({"username": document.fakeLogin.username.value,
                                    "password": document.fakeLogin.password.value}),
        success: function(data, textStatus, request) {
            window.localStorage.setItem("X-CSRF-Token", request.getResponseHeader('X-CSRF-Token'));
            window.open("/filer", "_self")
        },
        error: function (request, textStatus, errorThrown) {
            alert("Error!" + request.responseText);
        }
    });
    return false;
}