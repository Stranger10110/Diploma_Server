function createDropAreaHandlers() {
    const dropArea = document.getElementById("drop-area");

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


function normalizePath(path) {
    return path.replace(/^[^a-z]+|[^\w:.-]+/gi, '').replace('.', '').replace('\n', '')
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
        case "xls": case "ods":
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
				    <i class="fas fa-code-branch hidden-file-btn" onclick="versionClicked(this);"></i>
                    <i class="far fa-share-square hidden-file-btn" onclick="shareClicked(this);"></i>
                    <i class="far fa-trash-alt hidden-file-btn" onclick="deleteClicked(this);"></i>
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
					<div class="file link-alike" onclick="folderClicked(this);">${folder_name}</div>
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


function createNewFolder() {
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
    const normalizedFilename = normalizePath(currentFilerPath() + name)
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


function setModalContent(element_id, data, buttons, titleAppend) {
    const modal = $(element_id)
    const dom_elem = modal.get(0)
    modal.html('[data]')
    dom_elem.innerHTML = dom_elem.innerHTML.replace('[data]', data)
    modal.dialog('option', 'buttons', buttons);
    modal.dialog('option', 'title', `${modal.dialog("option", "title")} ${titleAppend}`);
    modal.dialog("open")
    return modal
}

/**
 * Creates a json object including fields in the form
 *
 * @param {HTMLElement} form The form element to convert
 * @return {Object} The form data
 */
const getFormJSON = (form) => {
    const data = new FormData(form);
    return Array.from(data.keys()).reduce((result, key) => {
        if (result[key] !== undefined) {
            result[key] = data.getAll(key)
            return result
        }
        result[key] = data.get(key);
        return result;
    }, {});
};

function getFileName(path) {
    return path.split('\\').pop().split('/').pop();
}

function createLink(relPath, event) {
    let send = true
    const data = getFormJSON(event.target)
    if (data.type[0] === 'group' && data.type[1] === '') {
        alert("Пожалуйста, укажите название группы доступа")
        send = false
    }
    const exp_time = Date.parse(data.params[1])
    if (data.params[0] === 'on' && (isNaN(exp_time) || exp_time <= new Date().getTime())) {
        alert("Пожалуйста, укажите время жизни ссылки (позднее текущей даты)")
        send = false
    }

    if (send) {
        const csrf_token = getCsrfToken()
        const json = {
            'path': relPath,
            'exp_time': data.params[0] === 'on' ? Date.parse(data.params[1]).toString(10) : '0',
            'type': data.type[0] === 'group' ? 'group_' + data.type[1] : 'public',
            'permission': (data.params.length === 2 && data.params[1] === 'on')
                            || (data.params.length === 3 && data.params[2] === 'on') ? 'rw' : 'r'
        }

        $.ajax({
            type: 'PUT',
            url: '/api/shared_link',
            headers: {'X-CSRF-Token': csrf_token},
            processData: false,
            contentType: "application/json",
            data: JSON.stringify(json),

            success: function(data, textStatus, request) {
                const link = `${window.location.host}/share/${data.link}`
                const html = `
                    <div>
                        Ссылка (${data.type.slice(0) === 'p' ? 'публичная' : 'для группы ' + data.type.slice(2, )}):
                        <a href="${window.location.protocol}//${link}" target="_blank"> ${link} </a>
                        <i class="far fa-trash-alt clickable" style="margin-left: 5px;" onclick="removeSharedLink(this);"></i>
                    </div>`

                $("#share-dialog").dialog('option', 'title', 'Поделиться')
                setModalContent("#share-dialog", html, {}, getFileName(relPath))
            },
            error: function (request, textStatus, errorThrown) {
                handleRequestError(request)
            },
        });
    }
}

function toIsoString(date) {
    const tzo = -date.getTimezoneOffset(),
        dif = tzo >= 0 ? '+' : '-',
        pad = function(num) {
            const norm = Math.floor(Math.abs(num));
            return (norm < 10 ? '0' : '') + norm;
        };

    return date.getFullYear() +
        '-' + pad(date.getMonth() + 1) +
        '-' + pad(date.getDate()) +
        'T' + pad(date.getHours()) +
        ':' + pad(date.getMinutes()) +
        ':' + pad(date.getSeconds()); // +
        // dif + pad(tzo / 60) +
        // ':' + pad(tzo % 60);
}

function normFilename(filename) {
    return filename.replace('\n', '').replaceAll(' ', '')
}

function shareClicked(obj) {
    const csrf_token = getCsrfToken()
    const relPath = currentFilerPath() + normFilename(obj.parentNode.parentElement.innerText)

    if (window._share_link_params_form == null) {
        window._share_link_params_form = `
            <form id="link-share-params">
                <fieldset>
                    <legend>Тип</legend>
                    
                    <input type="radio" name="type" id="link-type-public" value="public" class="ui-widget-content ui-corner-all" checked>
                    <label for="link-type">Публичная ссылка</label>
               
                    <input type="radio" name="type" id="link-type-group" value="group" class="ui-widget-content ui-corner-all" style="margin-left: 20px;">
                    <label for="link-type-group">Для группы</label>
                    <input type="text" name="type" id="link-type-group-name" placeholder="название" class="ui-widget-content ui-corner-all">
                </fieldset>
                
                <fieldset style="margin-top: 10px;">
                    <legend>Параметры</legend>
                    
                    <input type="checkbox" name="params" id="ttl" class="ui-widget-content ui-corner-all">
                    <label for="ttl">Время жизни (до)</label>
                    <input type="datetime-local" name="params" id="ttl-value" min="${toIsoString(new Date())}" class="ui-widget-content ui-corner-all">
                    
                    <input type="checkbox" name="params" id="write-permission" style="margin-left: 20px;" class="ui-widget-content ui-corner-all">
                    <label for="write-permission">Разрешить запись?</label>
                 </fieldset>
            </form>`
    }


    $.ajax({
        type: 'GET',
        url: '/api/shared_link/' + relPath,
        headers: {'X-CSRF-Token': csrf_token},

        success: function(data, textStatus, request) {
            const link = `${window.location.host}/share/${data.link}`
            const html = `
                <div>
                    Ссылка (${data.type.slice(0) === 'p' ? 'публичная' : 'для группы ' + data.type.slice(2, )}):
                    <a href="${window.location.protocol}//${link}" target="_blank"> ${link} </a>
                    <i class="far fa-trash-alt clickable" style="margin-left: 5px;" onclick="removeSharedLink(this);"></i>
                </div>`

            setModalContent("#share-dialog", html, {}, obj.parentNode.parentElement.innerText)
        },
        error: function (request, textStatus, errorThrown) {
            if (request.status === 404) {
                setModalContent("#share-dialog", window._share_link_params_form,
                                     window._share_dialog_buttons, obj.parentNode.parentElement.innerText)
                document.querySelector('#link-share-params').addEventListener('submit', function (event) {
                    createLink(relPath, event)
                });
            } else {
                handleRequestError(request)
            }
        },
    });
}


function downgradeFileToVersion(relPath, event) {
    const csrf_token = getCsrfToken()
    const version = getFormJSON(event.target).version
    let json = {}

    if (version !== undefined) {
        json = {
            "version": parseInt(version, 10),
            "rel_path": relPath
        }
    } else {
        return
    }

    $.ajax({
        type: 'PATCH',
        url: '/api/version',
        headers: {'X-CSRF-Token': csrf_token},
        processData: false,
        contentType: "application/json",
        data: JSON.stringify(json),

        success: function(data, textStatus, request) {
            const newVersion = event.target.childElementCount + 1
            const html = `<div><input type="radio" name="version" id="version-${newVersion}" value="${newVersion}" class="ui-widget-content ui-corner-all"> ${newVersion}) текущая </div>`
            document.querySelector('#file-versions').innerHTML += html
        },
        error: function (request, textStatus, errorThrown) {
            handleRequestError(request)
        },
    });
}

function versionClicked(obj) {
    const csrf_token = getCsrfToken()
    const relPath = currentFilerPath() + normFilename(obj.parentNode.parentElement.innerText)

    $.ajax({
        type: 'GET',
        url: '/api/version/' + currentFilerPath() + obj.parentNode.parentElement.innerText,
        headers: {'X-CSRF-Token': csrf_token},

        success: function(data, textStatus, request) {
            if (data.versions.length > 0) {
                let versions = ''
                for (const v of data.versions) {
                    const split = v.split(';')
                    versions += `<div><input type="radio" name="version" id="version-${split[0]}" value="${split[0]}" class="ui-widget-content ui-corner-all"> ${split[0]}) ${split[1]} </div>`
                }
                const html = `
                <form id="file-versions" onsubmit="downgradeFileToVersion(${relPath})">
                    ${versions}
                </form>`

                setModalContent("#version-dialog", html, window._version_dialog_buttons, obj.parentNode.parentElement.innerText)

                document.querySelector('#file-versions').addEventListener('submit', function (event) {
                    downgradeFileToVersion(relPath, event)
                });
            } else {
                setModalContent("#version-dialog", 'Пока что это единственная версия файла', {}, obj.parentNode.parentElement.innerText)
            }
        },
        error: function (request, textStatus, errorThrown) {
            if (request.status === 404) {
                setModalContent("#share-dialog", window._share_link_params_form,
                    window._share_dialog_buttons, obj.parentNode.parentElement.innerText)
                document.querySelector('#link-share-params').addEventListener('submit', function (event) {
                    createLink(relPath, event)
                });
            } else {
                handleRequestError(request)
            }
        },
    });
}


function initDialogs() {
    const share = $("#share-dialog")
    const version = $("#version-dialog")
    const settings = {
        autoOpen: false,
        show: {effect: "slide", direction: 'up', duration: 200},
        hide: {effect: "slide", direction: 'up', duration: 200},
        closeOnEscape: true,
        draggable: false,
        resizable: false,
        modal: true,
        minHeight: 20,
        height:'auto',
        width:'auto',
        position: { my: "center", at: "center", of: $("#Filer-table") }
    }

    share.dialog(Object.assign(settings,
        {
            close: function() {
            // document.querySelector('#link-share-params').reset();
            // dialog.html('[data]')
            share.dialog('option', 'title', 'Поделиться')
            }
        }
    ));

    version.dialog(Object.assign(settings, {
        close: function() { version.dialog('option', 'title', 'Версии') }
    }));

    window._share_dialog_buttons = {
        "Отмена": function() {
            share.dialog("close");
        },
        "Получить ссылку": function () {
            const event = new Event('submit', {
                'bubbles'    : true, // Whether the event will bubble up through the DOM or not
                'cancelable' : true  // Whether the event may be canceled or not
            });
            document.querySelector('#link-share-params').dispatchEvent(event);
        }
    }

    window._version_dialog_buttons = {
        "Отмена": function() {
            version.dialog("close");
        },
        "Откатить до выбранной версии": function () {
            const event = new Event('submit', {
                'bubbles'    : true, // Whether the event will bubble up through the DOM or not
                'cancelable' : true  // Whether the event may be canceled or not
            });
            document.querySelector('#file-versions').dispatchEvent(event);
        }
    }
}


function removeSharedLink(obj) {
    const csrf_token = getCsrfToken()
    const json = {
        'path': currentFilerPath() + normFilename($("#share-dialog").dialog('option', 'title').split(' ')[1]),
        'link': normFilename(obj.parentNode.children[0].innerText.split('share/')[1])
    }

    $.ajax({
        type: 'DELETE',
        url: '/api/shared_link',
        headers: {'X-CSRF-Token': csrf_token},
        processData: false,
        contentType: "application/json",
        data: JSON.stringify(json),

        success: function(data, textStatus, request) {
            $("#share-dialog").dialog('close')
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
        initDialogs()
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