<#import "template.ftl" as layout>
<@layout.registrationLayout displayMessage=!messagesPerField.existsError('username') displayInfo=true; section>

    <#if section = "header">
        <div class="fulcrum-logo">
            <img src="${url.resourcesPath}/img/fulcrum-logo.svg" alt="Fulcrum" width="210" height="48" />
        </div>
        ${msg("emailForgotTitle")}
    <#elseif section = "form">
        <div id="kc-form">
            <div id="kc-form-wrapper">
                <form id="kc-reset-password-form" action="${url.loginAction}" method="post" style="display:flex;flex-direction:column;gap:1rem;">

                    <div class="fulcrum-field">
                        <label for="username">
                            <#if !realm.loginWithEmailAllowed>${msg("username")}
                            <#elseif !realm.registrationEmailAsUsername>${msg("usernameOrEmail")}
                            <#else>${msg("email")}
                            </#if>
                        </label>
                        <input id="username" name="username" type="text"
                               value="${(auth.attemptedUsername!'')}"
                               placeholder="<#if !realm.loginWithEmailAllowed>${msg("username")}<#elseif !realm.registrationEmailAsUsername>${msg("usernameOrEmail")}<#else>${msg("email")}</#if>"
                               autofocus autocomplete="username"
                               aria-invalid="<#if messagesPerField.existsError('username')>true</#if>" />

                        <#if messagesPerField.existsError('username')>
                            <span class="fulcrum-error" aria-live="polite">
                                ${kcSanitize(messagesPerField.getFirstError('username'))?no_esc}
                            </span>
                        </#if>
                    </div>

                    <button type="submit" class="fulcrum-submit">
                        ${msg("doSubmit")}
                    </button>
                </form>
            </div>
        </div>
    <#elseif section = "info">
        <div class="fulcrum-register">
            <a href="${url.loginUrl}">${kcSanitize(msg("backToLogin"))?no_esc}</a>
        </div>
    </#if>

</@layout.registrationLayout>
