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

function uploadFile(file, i) {
    const csrf_token = getCsrfToken()
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
            alert(file.name + " успешно загружен")
            updateFileInfo(file)
            document.querySelector('#upload-button').value = ''
        },
        error: function (request, textStatus, errorThrown) {
            handleAuthError(request)
            document.querySelector('#upload-button').value = ''
        }
    })
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
    $.ajax({
        type: 'GET',
        url: '/secure/filer/' + path,
        headers: {
            'Accept': 'text/html',
            'Authorization': "Bearer " + csrf_token
        },
        success: function (data, textStatus, request) {
            insertListingInPage(data)
        },
        error: function (request, textStatus, errorThrown) {
            handleAuthError(request)
        }
    });
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
    openFolder(filer_path + '/')
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

    const data = `<tr><td><div class="file link-alike" onclick="return downloadFile(this);">${file.name}</div></td><td>${size}</td><td>${date}</td></tr>`
    insertFileInfoInPage(data, file.name)
}


function downloadFile(obj) {
    const csrf_token = getCsrfToken()
    $.ajax({
        type: 'GET',
        url: '/api/filer/' + currentFilerPath() + obj.innerText,
        dataType: 'binary',
        headers: {'X-CSRF-Token': csrf_token},
        processData: false,
        success: function (blob) {
            const windowUrl = window.URL || window.webkitURL;
            const url = windowUrl.createObjectURL(blob);
            const anchor = document.querySelector("#download-file")
            anchor.setAttribute('href', url);
            anchor.setAttribute('download', obj.innerText);
            anchor.click();
            windowUrl.revokeObjectURL(url);
        },
        error: function (request, textStatus, errorThrown) {
            handleAuthError(request)
        }
    });
}


function deleteClicked(obj) {
    let del = true
    let folder = false
    const name = obj.parentElement.innerText

    if (obj.parentElement.querySelector("i.fa-folder") != null) { // is folder
        folder = true
        if (!window.confirm(`Все файлы и вложенные папки в ${currentFilerPath()}/${name} будут удалены. Продолжить?`)) {
            del = false
        }
    }

    if (del) {
        const csrf_token = getCsrfToken()
        $.ajax({
            type: 'DELETE',
            url: '/api/filer/' + currentFilerPath() + name,
            headers: {'X-CSRF-Token': csrf_token},
            success: function(data, textStatus, request) {
                if (folder) {
                    alert(`Папка ${name} была успешно удалена`)
                } else {
                    alert(`Файл ${name} был успешно удален`)
                }
                const temp = obj.parentNode.parentNode.parentNode
                temp.parentNode.removeChild(temp)
            },
            error: function (request, textStatus, errorThrown) {
                handleAuthError(request)
            }
        });
    }
}


$(document).ready(function () {
    openFolder('')
    makePageFancy()
});

// Update CSRF token after each request
$(document).ajaxComplete(function(event, request, settings) {
    const token = request.getResponseHeader('X-CSRF-Token');
    if (token != null) {
        window.localStorage.setItem("X-CSRF-Token", token);
    }
});

