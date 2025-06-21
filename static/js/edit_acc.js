function addIRouteRouteInput() {
    $('#irouteRoutesContainer').append(`
            <div class="input-group route-input-group">
                <input type="text" class="form-control irouteRouteInput" placeholder="10.8.0.0 255.255.255.0" required>
                <button class="btn btn-outline-danger" type="button" onclick="removeRouteInput(this)">删除</button>
            </div>
        `);
}

function addNormalRouteInput() {
    $('#normalRoutesContainer').append(`
            <div class="input-group route-input-group">
                <input type="text" class="form-control normalRouteInput" placeholder="192.168.1.0 255.255.255.0" required>
                <button class="btn btn-outline-danger" type="button" onclick="removeRouteInput(this)">删除</button>
            </div>
        `);
}

function removeRouteInput(button) {
    $(button).closest('.input-group').remove();
}

$(document).ready(function () {
    // 初始化Select2
    $('#templateId, #iroutes').select2({
        theme: 'bootstrap-5',
        placeholder: '请选择...',
        allowClear: true
    });
});
// 验证函数
function validateEmail(email) {
    const re = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
    return re.test(email);
}

function validatePhone(phone) {
    const re = /^1[3-9]\d{9}$/;
    return re.test(phone);
}

function validateRoute(route) {
    const re = /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3} \d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/;
    return re.test(route);
}

function showError(elementId, message) {
    const element = document.getElementById(elementId);
    element.textContent = message;
    element.style.display = 'block';
}
// 表单验证和提交
$('#editAccountForm').submit(function (e) {
    e.preventDefault();
    $('.error-message').hide();

    // 获取表单数据
    const formData = {
        username: $('#usernameHidden').val(),
        password: $('#password').val(),
        display_name: $('#displayName').val().trim(),
        email: $('#email').val().trim(),
        phone: $('#phone').val().trim(),
        is_iroute: $('#irouteType').is(':checked'),
        enabled: $('#enabled').is(':checked'),
        template_ids: $('#templateId').val() || [],
        iroute_ids: $('#iroutes').val() || [],
        routes: []
    };

    // 验证必填字段
    let isValid = true;
    if (!formData.display_name) {
        showError('displayNameError', '中文名称不能为空');
        isValid = false;
    }
    if (!formData.email) {
        showError('emailError', '邮箱不能为空');
        isValid = false;
    } else if (!validateEmail(formData.email)) {
        showError('emailError', '邮箱格式不正确');
        isValid = false;
    }
    if (!formData.phone) {
        showError('phoneError', '手机号不能为空');
        isValid = false;
    } else if (!validatePhone(formData.phone)) {
        showError('phoneError', '手机号格式不正确');
        isValid = false;
    }

    // 收集并验证路由
    const routeSelector = formData.is_iroute ? '.irouteRouteInput' : '.normalRouteInput';
    const errorElement = formData.is_iroute ? 'irouteRouteError' : 'normalRouteError';

    $(routeSelector).each(function () {
        const route = $(this).val().trim();
        if (route) {
            if (!validateRoute(route)) {
                showError(errorElement, '路由格式不正确，应为 "IP地址 子网掩码"');
                isValid = false;
            }
            formData.routes.push({route: route});
        }
    });

    if (formData.routes.length === 0) {
        showError(errorElement, '至少需要添加一个路由');
        isValid = false;
    }

    if (!isValid) return;

    // 提交数据
    fetch('/api/accounts', {
        method: 'PUT',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(formData)
    })
        .then(response => response.json())
        .then(data => {
            if (data.error) {
                alert('保存失败: ' + data.error);
            } else {
                alert('账号更新成功');
                window.location.href = '/accounts';
            }
        })
        .catch(error => {
            alert('保存失败: ' + error.message);
        });
});

document.querySelector('.btn-secondary').addEventListener('click', function (e) {
    if (!confirm('确定要取消编辑吗？所有未保存的更改将会丢失。')) {
        e.preventDefault();
    }
});