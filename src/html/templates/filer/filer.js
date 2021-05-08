let dropArea = document.getElementById("drop-area");

['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
    dropArea.addEventListener(eventName, preventDefaults, false)
    document.body.addEventListener(eventName, preventDefaults, false)
});

['dragenter', 'dragover'].forEach(eventName => {
    dropArea.addEventListener(eventName, highlight, false)
});

['dragleave', 'drop'].forEach(eventName => {
    dropArea.addEventListener(eventName, unhighlight, false)
});


dropArea.addEventListener('drop', handleDrop, false)

function preventDefaults(e) {
    e.preventDefault()
    e.stopPropagation()
}

function highlight(e) {
    dropArea.classList.add('highlight')
}

function unhighlight(e) {
    dropArea.classList.remove('highlight')
}

function handleDrop(e) {
    const dt = e.dataTransfer;
    const files = dt.files;
    handleFiles(files)
}

function handleFiles(files) {
    files = [...files]
    files.forEach(uploadFile)
}


function currentFilerPath() {
    let filer_path = ""
    document.querySelector("#current-path").childNodes.forEach(path => {
        if (path.tagName === 'DIV') {
            filer_path += path.innerText
        }
    });
    return filer_path
}

function getCsrfToken() {
    const csrf_token = window.localStorage.getItem("X-CSRF-Token")
    if (csrf_token == null) {
        window.open("/login", "_self")
    }
    return csrf_token
}

function handleAuthError(request) {
    alert("Error!" + request.responseText);
    if (request.status === 403) {
        window.open("/login", "_self")
    }
}

function uploadFile(file, i) {
    const csrf_token = getCsrfToken()
    if (csrf_token != null) {
        let formData = new FormData()
        formData.append('file', file)

        $.ajax({
            type: 	'POST',
            url: 	'/api/filer/' + currentFilerPath() + file.name,
            headers: {
                'Accept': 'text/html',
                'Authorization': "Bearer " + csrf_token
            },
            data: formData,
            contentType: false,
            processData: false,

            success: function(data, textStatus, request) {
                window.localStorage.setItem("X-CSRF-Token", request.getResponseHeader('X-CSRF-Token'));
                alert(file.name + " успешно загружен")
                updateFileInfo(file)
                document.querySelector('#upload-button').value = ''
            },
            error: function (request, textStatus, errorThrown) {
                handleAuthError(request)
                document.querySelector('#upload-button').value = ''
            }
        })
    } else {
        window.open("/login", "_self")
    }
    return false;
}


function makePageFancy() {
    $('#Filer-filter').on('propertychange input', function(e)
    {
        $('.Filer-no-results').remove();
        var $this = $(this);
        var search = $this.val().toLowerCase();
        var $target = $('#Filer');
        var $rows = $target.find('tbody tr');
        if (search === '')
        {
            $rows.removeClass('filter-hide');
            // buildNav();
            // paginate();
        }
        else
        {
            $rows.each(function()
            {
                var $this = $(this);
                $this.text().toLowerCase().indexOf(search) === -1 ? $this.addClass('filter-hide') : $this.removeClass('filter-hide');
            })
            // buildNav();
            // paginate();
            if ($target.find('tbody tr:visible').size() === 0)
            {
                var col_span = $target.find('tr').first().find('td').size();
                var no_results = $('<tr class="Filer-no-results"><td colspan="'+col_span+'">No results found</td></tr>');
                $target.find('tbody').append(no_results);
            }
        }
    });
    $('.panel-heading span.filter').on('click', function(e)
    {
        var $this = $(this),
            $panel = $this.parents('.panel');
        $panel.find('.panel-body').slideToggle({duration: 200}, function()
        {
            if($this.css('display') !== 'none')
            {
                $panel.find('.panel-body input').focus();
            }
        });
    });
}


function insertListingInPage(data) {
    const split = data.split('^^^')
    document.querySelector("tbody").innerHTML = split[0]
    const current_path_elem = document.querySelector("#current-path")
    current_path_elem.innerHTML = split[1]

    const paths = current_path_elem.children
    const back_button = document.querySelector("#back-button")
    if (paths.length > 1) {
        const onclick = paths.item(paths.length - 2).getAttribute('onclick')
        back_button.setAttribute('onclick', onclick)
        back_button.innerHTML = '<i class="fas fa-arrow-left "></i> Назад'
        document.querySelector("#path-root").innerHTML = './'
    } else {
        back_button.innerHTML = 'Filer'
        document.querySelector("#path-root").innerHTML = ""
    }
}

function openFolder(path) {
    const csrf_token = getCsrfToken()
    if (csrf_token != null) {
        return $.ajax({
            type: 'GET',
            url: '/secure/filer/' + path,
            headers: {
                'Accept': 'text/html',
                'Authorization': "Bearer " + csrf_token
            },
            success: function (data, textStatus, request) {
                window.localStorage.setItem("X-CSRF-Token", request.getResponseHeader('X-CSRF-Token'));
                insertListingInPage(data)
            },
            error: function (request, textStatus, errorThrown) {
                handleAuthError(request)
            }
        });
    } else {
        window.open("/login", "_self")
    }
}

function folderClicked(obj) {
    if (obj === "#"){
        return
    } else if (typeof obj === "string") {
        openFolder(obj)
        return
    }

    let filer_path = currentFilerPath()
    filer_path = filer_path.slice(2)
    if (filer_path === '') {
        filer_path = obj.innerText
    } else {
        filer_path += '/' + obj.innerText
    }
    openFolder(filer_path)
    return 0
}


function insertFileInfoInPage(data, filename) {
    const xpath = `//tr[.//text()[contains(., '${filename}')]]`;
    const matchingElement = document.evaluate(xpath, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
    if (matchingElement != null) {
        matchingElement.outerHTML = data
    } else {
        document.querySelector("#Filer-table").innerHTML += '\n' + data
    }
}

function updateFileInfo(file) {
    const d = new Date();
    const date = `${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.${d.getFullYear()}, ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
    let size = (file.size / 1048576).toFixed(2) + 'MB'
    if (size === '0.00MB') {
        size = (file.size / 1024).toFixed(2) + 'KB'
    }

    const data = `<tr><td><div class="file link-alike" onclick="return fileClicked(this);">${file.name}</div></td><td>${size}</td><td>${date}</td></tr>`
    insertFileInfoInPage(data, file.name)
}


$(document).ready(function () {
    openFolder('')
    makePageFancy()
});

