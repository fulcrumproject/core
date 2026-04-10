<#import "template.ftl" as layout>
<@layout.registrationLayout displayMessage=!messagesPerField.existsError('username','password') displayInfo=realm.password && realm.registrationAllowed && !registrationDisabled??; section>

    <#if section = "header">
        <div class="fulcrum-logo">
            <img src="${url.resourcesPath}/img/fulcrum-logo.svg" alt="Fulcrum" width="210" height="48" />
        </div>
        ${msg("loginAccountTitle")}
    <#elseif section = "form">
        <div id="kc-form">
            <div id="kc-form-wrapper">
                <form id="kc-form-login" onsubmit="login.disabled = true; return true;" action="${url.loginAction}" method="post">

                    <#if !usernameHidden??>
                        <div class="fulcrum-field">
                            <label for="username">
                                <#if !realm.loginWithEmailAllowed>${msg("username")}
                                <#elseif !realm.registrationEmailAsUsername>${msg("usernameOrEmail")}
                                <#else>${msg("email")}
                                </#if>
                            </label>
                            <input id="username" name="username" type="text"
                                   value="${(login.username!'')}"
                                   placeholder="<#if !realm.loginWithEmailAllowed>${msg("username")}<#elseif !realm.registrationEmailAsUsername>${msg("usernameOrEmail")}<#else>${msg("email")}</#if>"
                                   autofocus autocomplete="username"
                                   aria-invalid="<#if messagesPerField.existsError('username','password')>true</#if>" />

                            <#if messagesPerField.existsError('username','password')>
                                <span class="fulcrum-error" aria-live="polite">
                                    ${kcSanitize(messagesPerField.getFirstError('username','password'))?no_esc}
                                </span>
                            </#if>
                        </div>
                    </#if>

                    <div class="fulcrum-field">
                        <label for="password">${msg("password")}</label>
                        <div class="fulcrum-input-group">
                            <input id="password" name="password" type="password"
                                   placeholder="${msg("password")}"
                                   autocomplete="current-password"
                                   aria-invalid="<#if messagesPerField.existsError('username','password')>true</#if>" />
                            <button class="fulcrum-password-toggle" type="button"
                                    aria-label="${msg('showPassword')}"
                                    aria-controls="password"
                                    data-password-toggle
                                    data-label-show="${msg('showPassword')}"
                                    data-label-hide="${msg('hidePassword')}">
                                <svg class="fulcrum-eye-open" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0"/><circle cx="12" cy="12" r="3"/></svg>
                                <svg class="fulcrum-eye-closed" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" style="display:none"><path d="M10.733 5.076a10.744 10.744 0 0 1 11.205 6.575 1 1 0 0 1 0 .696 10.747 10.747 0 0 1-1.444 2.49"/><path d="M14.084 14.158a3 3 0 0 1-4.242-4.242"/><path d="M17.479 17.499a10.75 10.75 0 0 1-15.417-5.151 1 1 0 0 1 0-.696 10.75 10.75 0 0 1 4.446-5.143"/><path d="m2 2 20 20"/></svg>
                            </button>
                        </div>
                    </div>

                    <div class="fulcrum-options">
                        <#if realm.rememberMe && !usernameHidden??>
                            <div class="fulcrum-remember">
                                <input id="rememberMe" name="rememberMe" type="checkbox"
                                       <#if login.rememberMe??>checked</#if> />
                                <label for="rememberMe">${msg("rememberMe")}</label>
                            </div>
                        </#if>
                        <#if realm.resetPasswordAllowed>
                            <a href="${url.loginResetCredentialsUrl}">${msg("doForgotPassword")}</a>
                        </#if>
                    </div>

                    <input type="hidden" id="id-hidden-input" name="credentialId"
                           <#if auth.selectedCredential?has_content>value="${auth.selectedCredential}"</#if> />

                    <button type="submit" id="kc-login" name="login" class="fulcrum-submit">
                        ${msg("doLogIn")}
                    </button>
                </form>
            </div>
        </div>

        <script>
            document.querySelector('[data-password-toggle]').addEventListener('click', function () {
                var input = document.getElementById(this.getAttribute('aria-controls'));
                var isPassword = input.type === 'password';
                input.type = isPassword ? 'text' : 'password';
                this.setAttribute('aria-label', isPassword ? this.dataset.labelHide : this.dataset.labelShow);
                this.querySelector('.fulcrum-eye-open').style.display = isPassword ? 'none' : '';
                this.querySelector('.fulcrum-eye-closed').style.display = isPassword ? '' : 'none';
            });
        </script>
    <#elseif section = "info">
        <#if realm.password && realm.registrationAllowed && !registrationDisabled??>
            <div class="fulcrum-register">
                <span>${msg("noAccount")}</span>
                <a href="${url.registrationUrl}">${msg("doRegister")}</a>
            </div>
        </#if>
    <#elseif section = "socialProviders">
        <#if realm.password && social?? && social.providers?has_content>
            <div class="fulcrum-social">
                <div class="fulcrum-social-divider">
                    <span>${msg("identity-provider-login-label")}</span>
                </div>
                <ul class="fulcrum-social-list">
                    <#list social.providers as p>
                        <li>
                            <a id="social-${p.alias}" href="${p.loginUrl}" class="fulcrum-social-btn">
                                <#if p.iconClasses?has_content>
                                    <i class="${p.iconClasses!}" aria-hidden="true"></i>
                                </#if>
                                <span>${p.displayName!}</span>
                            </a>
                        </li>
                    </#list>
                </ul>
            </div>
        </#if>
    </#if>

</@layout.registrationLayout>
