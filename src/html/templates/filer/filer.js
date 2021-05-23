function createDropAreaHandlers() {
    window.dropArea = document.getElementById("drop-area");

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
}

function uploadFile(file, i) {
    if (file.size === 0) {
        return
    }

    const csrf_token = getCsrfToken()
    let formData = new FormData()
    formData.append('file', file)

    const progressBar = progressBarElement(file.name)
    document.querySelector("#uploads").innerHTML += progressBar[0]
    downloadsUploadsUI()

    const removeElement = function() {
        const progressBarElem = document.querySelector("#progress-" + progressBar[1])
        progressBarElem.parentNode.parentNode.removeChild(progressBarElem.parentNode)
    }

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

        xhr: function () {
            let xhr = new XMLHttpRequest()
            xhr.upload.addEventListener('progress', function(e) {
                const progress = (e.loaded / e.total) * 100
                if (e.loaded % 2 === 0) {
                    document.querySelector("#progress-" + progressBar[1]).style.width = `${progress}%`;
                }
            });

            document.querySelector("#progress-" + progressBar[1] + "-cancel").addEventListener("click", function () {
                xhr.abort()
                window.xhr_aborted = 1
            });

            return xhr
        },

        success: function(data, textStatus, request) {
            // alert(file.name + " успешно загружен")
            updateFileInfo(file)
            document.querySelector('#upload-button').value = ''
        },
        error: function (request, textStatus, errorThrown) {
            if (window.xhr_aborted !== 1) {
                handleRequestError(request)
                window.xhr_aborted = 0
            }
            document.querySelector('#upload-button').value = ''
        },

        complete: removeElement
    })
}


function currentFilerPath() {
    let filer_path = ""
    document.querySelector("#current-path").childNodes.forEach(path => {
        if (path.tagName === 'DIV') {
            filer_path += path.innerText
        }
    });
    return filer_path.replace('./','')
}

function getCsrfToken() {
    const csrf_token = window.localStorage.getItem("X-CSRF-Token")
    if (csrf_token == null) {
        window.open("/login", "_self")
    }
    return csrf_token
}

function handleRequestError(request) {
    if (request.status === 403) {
        request.responseText = "Запрещено!"
    }
    alert("Error!" + '   ' + request.responseText);

    if (request.status === 401) { // StatusUnauthorized
        window.open("/login", "_self")
    } else if (request.status === 404) { // StatusNotFound
        window.open("/not_found_404", "_self")
    }
}


function makePageFancy() {
    $('#Filer-filter').on('propertychange input', function(e)
    {
        $('.Filer-no-results').remove();
        const $this = $(this);
        const search = $this.val().toLowerCase();
        const $target = $('#Filer');
        const $rows = $target.find('tbody tr');
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
                const $this = $(this);
                $this.text().toLowerCase().indexOf(search) === -1 ? $this.addClass('filter-hide') : $this.removeClass('filter-hide');
            })
            // buildNav();
            // paginate();
            if ($target.find('tbody tr:visible').size() === 0)
            {
                const col_span = $target.find('tr').first().find('td').size();
                const no_results = $('<tr class="Filer-no-results"><td colspan="' + col_span + '">No results found</td></tr>');
                $target.find('tbody').append(no_results);
            }
        }
    });
    $('.panel-heading span.filter').on('click', function(e)
    {
        const $this = $(this),
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


function modifyPageContent(data) {
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
    window.sessionStorage.setItem("current_path", path)

    $.ajax({
        type: 'GET',
        url: '/secure/filer/' + path,
        headers: {
            'Accept': 'text/html',
            'Authorization': "Bearer " + csrf_token
        },
        success: function(data, textStatus, request) {
            modifyPageContent(data)
        },
        error: function (request, textStatus, errorThrown) {
            handleRequestError(request)
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
    if (filer_path === '') {
        filer_path = obj.innerText
    } else {
        filer_path += obj.innerText
    }
    filer_path += '/'

    openFolder(filer_path)
    return 0
}


function insertFileInfoInPage(data, filename) {
    const xpath = `//tr[.//text()[contains(., '${filename}')]]`;
    const matchingElement = document.evaluate(xpath, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
    if (matchingElement != null) {
        matchingElement.outerHTML = data
    } else {
        document.querySelector("#Filer-table > tbody").innerHTML += '\n' + data
    }
}

function getCurrentDateString() {
    const d = new Date();
    return `${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.${d.getFullYear()}, ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

function updateFileInfo(file) {
    const date = getCurrentDateString()
    let size = (file.size / 1048576).toFixed(2) + 'MB'
    if (size === '0.00MB') {
        size = (file.size / 1024).toFixed(2) + 'KB'
    }

    let icon = ''
    switch (file.name.split('.').pop().slice(0, 3)) {
        case "doc": case "odt":
            icon = "<i class=\"far fa-file-word\"></i>"
            break
        case "pdf":
            icon = "<i class=\"far fa-file-pdf\"></i>"
            break
        case "txt":
            icon = "<i class=\"far fa-file-alt\"></i>"
            break
        case "xls":
            icon = "<i class=\"far fa-file-excel\"></i>"
            break
        case "csv":
            icon = "<i class=\"fas fa-file-csv\"></i>"
            break
        case "ppt":
            icon = "<i class=\"far fa-file-powerpoint\"></i>"
            break
        case "jpg": case "jpe": case "png": case "bmp":
            icon = "<i class=\"far fa-file-image\"></i>"
    }

    const data = `
		<tr> <td>
			<div class="with-file-actions">
				<div style="display:inline-flex; align-items:center;">
					${icon} <div class="file link-alike" onclick="downloadFile(this);"> ${file.name} </div>
				</div>

				<div class="file-actions">
                    <i class="far fa-share-square hidden-file-btn" onclick="shareClicked(this);"> </i>
                    <i class="far fa-trash-alt hidden-file-btn" onclick="deleteClicked(this);"> </i>
				</div>
			</div>
		</td>   <td> ${size} </td>   <td> ${date} </td> </tr>`
    insertFileInfoInPage(data, file.name)
}

function insertNewFolderInPage(folder_name) {
    const date = getCurrentDateString()
    const data = `
		<tr> <td>
			<div class="with-file-actions">
				<div style="display:inline-flex; align-items:center;">
					<i class="far fa-folder"></i>
					<div class="file link-alike" onclick="downloadFile(this);">${folder_name}</div>
				</div>
				
				<div class="file-actions">
                        <i class="far fa-share-square hidden-file-btn" onclick="shareClicked(this);"> </i>
                        <i class="far fa-trash-alt hidden-file-btn" onclick="deleteClicked(this);"> </i>
                </div>
			</div>
		</td>  <td></td>  <td> ${date} </td> </tr>`

    const row = document.querySelector("#Filer-table").tBodies[0].insertRow(0)
    row.innerHTML = data
}


function downloadFile(obj) {
    const csrf_token = getCsrfToken()

    const progressBar = progressBarElement(obj.innerText)
    document.querySelector("#downloads").innerHTML += progressBar[0]
    downloadsUploadsUI()

    const removeElement = function() {
        const progressBarElem = document.querySelector("#progress-" + progressBar[1])
        progressBarElem.parentNode.parentNode.removeChild(progressBarElem.parentNode)
    }

    $.ajax({
        type: 'GET',
        url: '/api/filer/' + currentFilerPath() + obj.innerText,
        dataType: 'binary',
        headers: {'X-CSRF-Token': csrf_token},
        processData: false,

        xhr: function () {
            let xhr = new XMLHttpRequest()
            xhr.addEventListener('progress', function(e) {
                const progress = (e.loaded / e.total) * 100
                if (e.loaded % 2 === 0) {
                    document.querySelector("#progress-" + progressBar[1]).style.width = `${progress}%`;
                }
            });

            document.querySelector("#progress-" + progressBar[1] + "-cancel").addEventListener("click", function () {
                xhr.abort()
                removeElement()
                downloadsUploadsUI()
            })

            return xhr
        },

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
            handleRequestError(request)
        },

        complete: removeElement
    });
}


function deleteClicked(obj) {
    let del = true
    let folder = false
    const name = obj.parentElement.parentElement.innerText.replace('\n', '')

    if (obj.parentElement.parentElement.querySelector("i.fa-folder") != null) { // is folder
        folder = true
        if (!window.confirm(`Все файлы и вложенные папки в ${currentFilerPath()}${name} будут удалены. Продолжить?`)) {
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
                // if (folder) {
                //     alert(`Папка ${name} была успешно удалена`)
                // } else {
                //     alert(`Файл ${name} был успешно удален`)
                // }
                const temp = obj.parentNode.parentNode.parentNode.parentNode
                temp.parentNode.removeChild(temp)
            },
            error: function (request, textStatus, errorThrown) {
                handleRequestError(request)
            }
        });
    }
}


function makeNewFolder() {
    let folder_name
    let xpath, N
    do {
        folder_name = prompt("Введите уникальное наименование", "Новая папка");
        xpath = `//tr[.//text()[normalize-space(.)='${folder_name}']]`  // `//tr[.//text()[contains(normalize-space(.), ${folder_name})]]`
        N = document.evaluate(xpath, document, null, XPathResult.UNORDERED_NODE_SNAPSHOT_TYPE, null).snapshotLength
    } while (folder_name != null && N !== 0)

    if (folder_name != null) {
        const csrf_token = getCsrfToken()
        $.ajax({
            type: 'PUT',
            url: '/api/filer/' + currentFilerPath() + folder_name + '/',
            headers: {'X-CSRF-Token': csrf_token},
            success: function(data, textStatus, request) {
                // alert(`Папка ${folder_name} была успешно создана`)
                insertNewFolderInPage(folder_name)
            },
            error: function (request, textStatus, errorThrown) {
                handleRequestError(request)
            }
        });
    }
}


function downloadsUploadsUI() {
    const downloads = document.querySelector("#downloads")
    if (downloads.childElementCount <= 1) {
        downloads.setAttribute("style", "display: none;")
    } else {
        downloads.setAttribute("style", "")
    }

    const uploads = document.querySelector("#uploads")
    if (uploads.childElementCount <= 1) {
        uploads.setAttribute("style", "display: none;")
    } else {
        uploads.setAttribute("style", "")
    }
}

function progressBarElement(name) {
    const normalizedFilename = (currentFilerPath() + name).replace(/^[^a-z]+|[^\w:.-]+/gi, '').replace('.', '')
    const html = `<div class="progress-container">
            <div class="clickable" id="progress-${normalizedFilename}-cancel">
                <i class="fas fa-times"></i>
            </div>
            <div class="text" style="margin-right:10px;">${name}</div>
            <div id="progress-${normalizedFilename}" class="progress-bar"></div>
        </div>`
    return [html, normalizedFilename]
}

function getCurrentFolder() {
    const path = window.sessionStorage.getItem("current_path")
    if (path == null) {
        return ''
    }
    return path
}


function shareClicked(obj) { // TODO: change a bit
    const csrf_token = getCsrfToken()

    const filename = obj.parentNode.parentElement.innerText.replace('\n', '').replaceAll(' ', '')
    $.ajax({
        type: 'GET',
        url: '/api/shared_link/' + currentFilerPath() + filename,
        headers: {'X-CSRF-Token': csrf_token},
        processData: false,

        success: function(data, textStatus, request) {
            console.log(data)
        },
        error: function (request, textStatus, errorThrown) {
            handleRequestError(request)
        },
    });
}

$(document).ready(function () {
    if (window.location.pathname.includes("filer/")) {
        openFolder(getCurrentFolder())
        makePageFancy()
        downloadsUploadsUI()
        createDropAreaHandlers()
    }
});

// Update CSRF token after each request
$(document).ajaxComplete(function(event, request, settings) {
    const token = request.getResponseHeader('X-CSRF-Token');
    if (token != null) {
        window.localStorage.setItem("X-CSRF-Token", token);
    }

    downloadsUploadsUI()
});