jQuery.cachedScript = function(url, options) {
    // Allow user to set any option except for dataType, cache, and url
    options = $.extend( options || {}, {
        dataType: "script",
        cache: true,
        url: url
    });

    // Use $.ajax() since it is more flexible than $.getScript
    // Return the jqXHR object so we can chain callbacks
    return jQuery.ajax(options);
};

function getHashLink() {
    return window.location.pathname.replaceAll('/share/', '')
}

function downloadZipFolder() {
    const csrf_token = getCsrfToken()

    const progressBar = progressBarElement('folder')
    document.querySelector("#downloads").innerHTML += progressBar[0]
    downloadsUploadsUI()

    const removeElement = function() {
        const progressBarElem = document.querySelector("#progress-" + progressBar[1])
        progressBarElem.parentNode.parentNode.removeChild(progressBarElem.parentNode)
    }

    const hashLink = getHashLink()

    $.ajax({
        type: 'GET',
        url: (hashLink.slice(-1) === 'a' ? '/api/public/zip/shared/filer/' : '/api/zip/shared/filer/') + hashLink + '/',
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
            anchor.setAttribute('download', 'folder.zip');
            anchor.click();
            windowUrl.revokeObjectURL(url);
        },
        error: function (request, textStatus, errorThrown) {
            handleRequestError(request)
        },

        complete: removeElement
    });
}

$(document).ready(function () {
    $.cachedScript( "/src/filer.js" ).done(function(script, textStatus) {
        uploadFile = function(file, i) {
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

            const hashLink = getHashLink()
            $.ajax({
                type: 	'POST',
                url: 	(hashLink.slice(-1) === 'a' ? '/api/public/shared/filer/' : '/api/shared/filer/') +
                        `${hashLink}/${currentFilerPath() + file.name}`,
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
                    }
                    document.querySelector('#upload-button').value = ''
                    window.xhr_aborted = 0
                },

                complete: removeElement
            })
        }

        getCsrfToken = function() {
            return window.localStorage.getItem("X-CSRF-Token")
        }

        modifyPageContent = function(data) {
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
                back_button.innerHTML = "Filer share"
                document.querySelector("#path-root").innerHTML = ""
            }

            const removeAllFileActions = function() {
                const parents = document.querySelectorAll(".with-file-actions")
                for (const parent of parents) {
                    parent.removeChild(parent.childNodes[3])
                }
            }

            if (split[2].length === 2) { // read only permission
                // Remove "Создать" и "Загрузить" buttons
                let parent = document.querySelector("#top-panel-actions")
                let children = parent.childNodes
                if (!(children[1] instanceof Text)) {
                    parent.removeChild(children[1])
                    parent.removeChild(children[2])
                }

                removeAllFileActions()

                // Disable drop area
                parent = document.querySelector("#drop-area")
                if (parent != null) {
                    parent.innerHTML = parent.childNodes[1].innerHTML
                    parent.removeAttribute("id")
                    parent.setAttribute("style", "border: 2px solid transparent;")
                }

                const child = document.querySelector("#upload-button")
                if (child != null) {
                    parent = child.parentNode
                    parent.removeChild(child)
                }

            } else if (split[2] === "rwf") { // read and write permission for a file
                // Remove "Создать" button
                const parent = document.querySelector("#top-panel-actions")
                parent.removeChild(parent.childNodes[1])

                removeAllFileActions()
                createDropAreaHandlers()

            } else { // read and write permission for a directory
                removeAllFileActions()
                createDropAreaHandlers()
            }

            // Set 'download zip' button
            if (document.querySelector('#Filer-table tbody').childElementCount > 1) {
                document.getElementsByClassName("panel")[0].outerHTML += `
                    <div class="'clickable" onclick="downloadZipFolder();" style="display: inline-flex">
                        <i class="far fa-file-archive"></i> Скачать папку (архив)
                    </div>
                `
            }
        }

        openFolder = function(path) {
            const csrf_token = getCsrfToken()
            window.sessionStorage.setItem("current_path", path)

            if (path.slice(0) !== '/') {
                path = '/' + path
            }

            const hashLink = getHashLink()
            $.ajax({
                type: 'GET',
                url: (hashLink.slice(-1) === 'a' ? '/shared/content/' : '/secure/shared/content/') +
                     `${getHashLink() + path}`,
                headers: {
                    'Accept': 'text/html',
                    'Authorization': "Bearer " + csrf_token
                },
                success: function (data, textStatus, request) {
                    modifyPageContent(data)
                },
                error: function (request, textStatus, errorThrown) {
                    handleRequestError(request)
                }
            });
        }

        updateFileInfo = function(file) {
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
			</div>
		</td>   <td> ${size} </td>   <td> ${date} </td> </tr>`
            insertFileInfoInPage(data, file.name)
        }

        insertNewFolderInPage = function(folder_name) {
            const date = getCurrentDateString()
            const data = `
		<tr> <td>
			<div class="with-file-actions">
				<div style="display:inline-flex; align-items:center;">
					<i class="far fa-folder"></i>
					<div class="file link-alike" onclick="folderClicked(this);">${folder_name}</div>
				</div>
			</div>
		</td>  <td></td>  <td> ${date} </td> </tr>`

            const row = document.querySelector("#Filer-table").tBodies[0].insertRow(0)
            row.innerHTML = data
        }

        downloadFile = function(obj) {
            const csrf_token = getCsrfToken()

            const progressBar = progressBarElement(obj.innerText)
            document.querySelector("#downloads").innerHTML += progressBar[0]
            downloadsUploadsUI()

            const removeElement = function() {
                const progressBarElem = document.querySelector("#progress-" + progressBar[1])
                progressBarElem.parentNode.parentNode.removeChild(progressBarElem.parentNode)
            }

            const hashLink = getHashLink()
            $.ajax({
                type: 'GET',
                url: (hashLink.slice(-1) === 'a' ? '/api/public/shared/filer/' : '/api/shared/filer/') +
                     `${hashLink}/${currentFilerPath() + obj.innerText}`,
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

        deleteClicked = function(obj) {
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
                const hashLink = getHashLink()
                $.ajax({
                    type: 'DELETE',
                    url: (hashLink.slice(-1) === 'a' ? '/api/public/shared/filer/' : '/api/shared/filer/') +
                         `${hashLink}/${currentFilerPath() + name}`,
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

        createNewFolder = function() {
            let folder_name
            let xpath, N
            do {
                folder_name = prompt("Введите уникальное наименование", "Новая папка");
                xpath = `//tr[.//text()[normalize-space(.)='${folder_name}']]`  // `//tr[.//text()[contains(normalize-space(.), ${folder_name})]]`
                N = document.evaluate(xpath, document, null, XPathResult.UNORDERED_NODE_SNAPSHOT_TYPE, null).snapshotLength
            } while (folder_name != null && N !== 0)

            if (folder_name != null) {
                const csrf_token = getCsrfToken()
                const hashLink = getHashLink()
                $.ajax({
                    type: 'PUT',
                    url: (hashLink.slice(-1) === 'a' ? '/api/public/shared/filer/' : '/api/shared/filer/') +
                         `${hashLink}/${currentFilerPath() + folder_name}`,
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

        openFolder(getCurrentFolder())
        makePageFancy()
        downloadsUploadsUI()
    })
});