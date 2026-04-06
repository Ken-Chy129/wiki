---
title: "Shiro 请求执行流程"
date: 2022-05-02T14:14:37+08:00
draft: false
summary: "**本文可能较长，但是通读一定能让你对整个shiro请求的执行流程有清晰的了解** 总体流程： 1、在过滤的过程中创建subject doFilter -SecurityManager -SubjectContext -创建subject -解析各种信息并赋值 2、若该subject未认证则进行认证..."
tags: [Shiro]
categories: [Java, Security]
source: csdn
source_id: "124540457"
source_url: "https://blog.csdn.net/qq_25046827/article/details/124540457"
---

**本文可能较长，但是通读一定能让你对整个shiro请求的执行流程有清晰的了解**


> 总体流程：
>
> 1、在过滤的过程中创建subject
>
> doFilter -> SecurityManager -> SubjectContext -> 创建subject -> 解析各种信息并赋值
>
> 2、若该subject未认证则进行认证并在认证时再次创建subject
>
> 调用realm中的doAuthenticationInfo()获得返回的信息重新创建subject并保存到session



### 一、AbstractShiroFilter


当我们使用shiro框架时，用户每次发送一个请求给服务端，都会被shiro的**AbstractShiroFilter过滤器所拦截**（AbstractShiroFilter是shiro的全局过滤器，所有的请求都会经过该过滤器）


```java
protected void doFilterInternal(ServletRequest servletRequest, ServletResponse servletResponse, final FilterChain chain) throws ServletException, IOException {
    Throwable t = null;

    try {
        final ServletRequest request = this.prepareServletRequest(servletRequest, servletResponse, chain);
        final ServletResponse response = this.prepareServletResponse(request, servletResponse, chain);
        Subject subject = this.createSubject(request, response);
        subject.execute(new Callable() {
            public Object call() throws Exception {
                AbstractShiroFilter.this.updateSessionLastAccessTime(request, response);
                AbstractShiroFilter.this.executeChain(request, response, chain);
                return null;
            }
        });
    } catch (ExecutionException var8) {
        t = var8.getCause();
    } catch (Throwable var9) {
        t = var9;
    }

    if (t != null) {
        if (t instanceof ServletException) {
            throw (ServletException)t;
        } else if (t instanceof IOException) {
            throw (IOException)t;
        } else {
            String msg = "Filtered request failed.";
            throw new ServletException(msg, t);
        }
    }
}
```


查看这个过滤器的**doFilterInternal()**方法，我们发现它主要做了两件事


- ```java
  // 创建一个subject
  createSubject(request, response)
  ```

- ```java
  // 将该subject绑定到当前线程，并更新会话的上次访问时间以及分发合适的过滤器
  subject.execute(new Callable() {
  	public Object call() throws Exception {
      	AbstractShiroFilter.this.updateSessionLastAccessTime(request, response);
      	AbstractShiroFilter.this.executeChain(request, response, chain);
      	return null;
      }
  });
  ```


### 二、createSubject(request, response)


我们先看createSubject(request, response)这个方法，追踪后来到以下方法


```java
protected WebSubject createSubject(ServletRequest request, ServletResponse response) {
    return (new Builder(this.getSecurityManager(), request, response)).buildWebSubject();
}
```


这个方法可以分为两部分


- new Builder(this.getSecurityManager(), request, response)
- buildWebSubject()


#### 1、new Builder(this.getSecurityManager(), request, response)


首先通过当前的安全管理器等**创建了一个Builder**，以下是其构造方法


```java
public Builder(SecurityManager securityManager, ServletRequest request, ServletResponse response) {
    super(securityManager);
    if (request == null) {
        throw new IllegalArgumentException("ServletRequest argument cannot be null.");
    } else if (response == null) {
        throw new IllegalArgumentException("ServletResponse argument cannot be null.");
    } else {
        this.setRequest(request);
        this.setResponse(response);
    }
}
```


首先**调用了父类的构造函数**，如下


```java
public Builder(SecurityManager securityManager) {
    if (securityManager == null) {
        throw new NullPointerException("SecurityManager method argument cannot be null.");
    } else {
        this.securityManager = securityManager;
        this.subjectContext = this.newSubjectContextInstance();
        if (this.subjectContext == null) {
            throw new IllegalStateException("Subject instance returned from 'newSubjectContextInstance' cannot be null.");
        } else {
            this.subjectContext.setSecurityManager(securityManager);
        }
    }
}
```


在其中设置了安全管理器，并**创建了一个subjectContext**，随后通过this.setRequest(request); this.setResponse(response);两个方法**为这个subjectContext设置request和response**，如下（response设置同理）


```java
protected WebSubject.Builder setRequest(ServletRequest request) {
    if (request != null) {
        ((WebSubjectContext)this.getSubjectContext()).setServletRequest(request);
    }

    return this;
}
```


至此Builder构造完成


#### 2、buildWebSubject()


追踪源码我们来到**WebSubject类的buildWebSubject()**方法


```java
public WebSubject buildWebSubject() {
    Subject subject = super.buildSubject();
    if (!(subject instanceof WebSubject)) {
        String msg = "Subject implementation returned from the SecurityManager was not a " + WebSubject.class.getName() + " implementation.  Please ensure a Web-enabled SecurityManager has been configured and made available to this builder.";
        throw new IllegalStateException(msg);
    } else {
        return (WebSubject)subject;
    }
}
```


其中调用了父类**Subject类的buildSubject()**方法


```java
public Subject buildSubject() {
    return this.securityManager.createSubject(this.subjectContext);
}
```


需要注意的是**这里的this.securityManager一般是DefaultWebSecurityManager类型**的，继承自DefaultSecurityManager类


其实最终调用的是**DefaultSecurityManager类的createSubject()方法**


```java
public Subject createSubject(SubjectContext subjectContext) {
    SubjectContext context = this.copy(subjectContext);
    context = this.ensureSecurityManager(context);
    context = this.resolveSession(context);
    context = this.resolvePrincipals(context);
    Subject subject = this.doCreateSubject(context);
    this.save(subject);
    return subject;
}
```


##### 1）this.copy(SubjectContext subjectContext)


这里copy方法用的是**DefaultWebSecurityManager重写的copy()**方法


```java
protected SubjectContext copy(SubjectContext subjectContext) {
    return (SubjectContext)(subjectContext instanceof WebSubjectContext ? new DefaultWebSubjectContext((WebSubjectContext)subjectContext) : super.copy(subjectContext));
}
```


将之前调用无参构造初始化的`SubjectContext`（上文构造Builder时使用this.newSubjectContextInstance()方法创建的，这个方法调用了DefaultSubjectContext的无参构造函数，实例化了一个SubjectContext）作为参数，**调用了`DefaultSubjectContext`的有参构造**，最终也**调用了`MapContext`中的有参构造**；返回了一个`SubjectContext`


> SubjectContext接口由DefaultSubjectContext实现（还有一个子类是DefaultWebSubjectContext），同时DefaultSubjectContext还继承自MapContext，其中有一个backingMap（本质也是一个map），里面是一路收集的一些信息，比如securityManger，subject，sessionId，principals，session等，key是DefaultSubjectContext中定义的一些常量（如下）。
>
> ```java
> private static final String SECURITY_MANAGER = DefaultSubjectContext.class.getName() + ".SECURITY_MANAGER";
> private static final String SESSION_ID = DefaultSubjectContext.class.getName() + ".SESSION_ID";
> private static final String AUTHENTICATION_TOKEN = DefaultSubjectContext.class.getName() + ".AUTHENTICATION_TOKEN";
> private static final String AUTHENTICATION_INFO = DefaultSubjectContext.class.getName() + ".AUTHENTICATION_INFO";
> private static final String SUBJECT = DefaultSubjectContext.class.getName() + ".SUBJECT";
> private static final String PRINCIPALS = DefaultSubjectContext.class.getName() + ".PRINCIPALS";
> private static final String SESSION = DefaultSubjectContext.class.getName() + ".SESSION";
> private static final String AUTHENTICATED = DefaultSubjectContext.class.getName() + ".AUTHENTICATED";
> private static final String HOST = DefaultSubjectContext.class.getName() + ".HOST";
> public static final String SESSION_CREATION_ENABLED = DefaultSubjectContext.class.getName() + ".SESSION_CREATION_ENABLED";
> public static final String PRINCIPALS_SESSION_KEY = DefaultSubjectContext.class.getName() + "_PRINCIPALS_SESSION_KEY";
> public static final String AUTHENTICATED_SESSION_KEY = DefaultSubjectContext.class.getName() + "_AUTHENTICATED_SESSION_KEY";
> private static final transient Logger log = LoggerFactory.getLogger(DefaultSubjectContext.class);
> ```
>
> 这个上下文的作用就是当初始化Subject时，从中获取需要的值来作为初始化Subject的参数
>
> resolve的意思是解析，接下来的三步就是解析上下文中的manager，session和principals，为`SubjectContext`也就是`MapContext`中的`backingMap`的key中添加相应的value；
>
> 点进去查看会发现调用了`DefaultSubjectContext`的父类`MapContext`中的`nullSafePut`与`put`方法；


##### 2）this.ensureSecurityManager(context)


用来确保上下文中已经存在securityManager，如果没有则将当前的securityManager设置进去


##### 3）this.resolveSession(context)


1. resolveSession(subjectContext)，首先尝试从context(MapContext)中获取session，没有就获取subject后尝试从subject中获取
2. 如果仍不存在则调用resolveContextSession(subjectContext)，试着从MapContext中获取sessionId
3. 根据sessionId实例化一个SessionKey对象，并通过SessionKey实例获取session
4. getSession(key)的任务直接交给sessionManager来执行
5. 如果key中获得的sessionId为null，则前往cookie中获取


具体如下：


```java
protected SubjectContext resolveSession(SubjectContext context) {
    if (context.resolveSession() != null) {
        log.debug("Context already contains a session.  Returning.");
        return context;
    } else {
        try {
            Session session = this.resolveContextSession(context);
            if (session != null) {
                context.setSession(session);
            }
        } catch (InvalidSessionException var3) {
            log.debug("Resolved SubjectContext context session is invalid.  Ignoring and creating an anonymous (session-less) Subject instance.", var3);
        }

        return context;
    }
}
```


如果上下文中有session则直接返回，没有则进行解析，**调用this.resolveContextSession(context)**方法


```java
protected Session resolveContextSession(SubjectContext context) throws InvalidSessionException {
    SessionKey key = this.getSessionKey(context);
    return key != null ? this.getSession(key) : null;
}
```


注意这里this.getSessionKey(context)是**调用DefaultWebSecurityManager类**中的方法


```java
protected SessionKey getSessionKey(SubjectContext context) {
    if (WebUtils.isWeb(context)) {
        Serializable sessionId = context.getSessionId();
        ServletRequest request = WebUtils.getRequest(context);
        ServletResponse response = WebUtils.getResponse(context);
        return new WebSessionKey(sessionId, request, response);
    } else {
        return super.getSessionKey(context);
    }
}
```


如果是web请求（携带着response和request）则context.getSessionId()这个接口会调用**DefaultSessionManager的实现**（注意此处不是DefaultWebSessionManager），如下：


```java
public Serializable getSessionId() {
    return getTypedValue(SESSION_ID, Serializable.class);
}
```


可以看这里其实时通过上下文去获取sessionId而不是获取请求中的sessionId


显然在此处上下文中还尚未有该信息，自然是获取不到的


随后获取request和response，根据sessionId（null）和request和response封装为WebSessionKey


------


回到`return key != null ? this.getSession(key) : null;`这里key!=null，所以调用this.getSession(key)


```java
public Session getSession(SessionKey key) throws SessionException {
    return this.sessionManager.getSession(key);
}
```


接着调用在AbstractNativeSessionManager中的实现


```java
public Session getSession(SessionKey key) throws SessionException {
    Session session = lookupSession(key);
    return session != null ? createExposedSession(session, key) : null;
}
```


```java
private Session lookupSession(SessionKey key) throws SessionException {
    if (key == null) {
        throw new NullPointerException("SessionKey argument cannot be null.");
    }
    return doGetSession(key);
}
```


这里key不等于null直接执行doGetSession(key)，来到AbstractvalidatingSessionMangager类的实现


```java
protected final Session doGetSession(final SessionKey key) throws InvalidSessionException {
    enableSessionValidationIfNecessary();

    log.trace("Attempting to retrieve session with key {}", key);

    Session s = retrieveSession(key);
    if (s != null) {
        validate(s, key);
    }
    return s;
}
```


接着执行`Session s = retrieveSession(key)`，来到DefaultSessionManager中的实现


```java
protected Session retrieveSession(SessionKey sessionKey) throws UnknownSessionException {
    Serializable sessionId = getSessionId(sessionKey);
    if (sessionId == null) {
        log.debug("Unable to resolve session ID from SessionKey [{}].  Returning null to indicate a " +
                "session could not be found.", sessionKey);
        return null;
    }
    Session s = retrieveSessionFromDataSource(sessionId);
    if (s == null) {
        //session ID was provided, meaning one is expected to be found, but we couldn't find one:
        String msg = "Could not find session with ID [" + sessionId + "]";
        throw new UnknownSessionException(msg);
    }
    return s;
}
```


其中第一句`Serializable sessionId = getSessionId(sessionKey);`由于我们在配置文件中设置的默认sessionManager是DefaultWebSessionManager，**所以这里执行的是DefaultWebSessionManager类中的getSessionId(sessionKey)而不是DefaultSessionManager类中的**，终于我们来到了下面这一步


```java
public Serializable getSessionId(SessionKey key) {
    Serializable id = super.getSessionId(key);
    if (id == null && WebUtils.isWeb(key)) {
        ServletRequest request = WebUtils.getRequest(key);
        ServletResponse response = WebUtils.getResponse(key);
        id = this.getSessionId(request, response);
    }

    return id;
}
```


这里第一句就是返回去调用DefaultSessionManager类中的方法， 然而这是通过上下文获取的，显然还是获取不到，接着id为空，我们来到this.getSessionId(request, response);


```java
protected Serializable getSessionId(ServletRequest request, ServletResponse response) {
    return this.getReferencedSessionId(request, response);
}
```


```java
private Serializable getReferencedSessionId(ServletRequest request, ServletResponse response) {
    String id = this.getSessionIdCookieValue(request, response);
    if (id != null) {
        request.setAttribute(ShiroHttpServletRequest.REFERENCED_SESSION_ID_SOURCE, "cookie");
    } else {
        id = this.getUriPathSegmentParamValue(request, "JSESSIONID");
        if (id == null && request instanceof HttpServletRequest) {
            String name = this.getSessionIdName();
            HttpServletRequest httpServletRequest = WebUtils.toHttp(request);
            String queryString = httpServletRequest.getQueryString();
            if (queryString != null && queryString.contains(name)) {
                id = request.getParameter(name);
            }

            if (id == null && queryString != null && queryString.contains(name.toLowerCase())) {
                id = request.getParameter(name.toLowerCase());
            }
        }

        if (id != null) {
            request.setAttribute(ShiroHttpServletRequest.REFERENCED_SESSION_ID_SOURCE, "url");
        }
    }

    if (id != null) {
        request.setAttribute(ShiroHttpServletRequest.REFERENCED_SESSION_ID, id);
        request.setAttribute(ShiroHttpServletRequest.REFERENCED_SESSION_ID_IS_VALID, Boolean.TRUE);
    }

    request.setAttribute(ShiroHttpServletRequest.SESSION_ID_URL_REWRITING_ENABLED, this.isSessionIdUrlRewritingEnabled());
    return id;
}
```


逐层调用最终来到


```java
private String getSessionIdCookieValue(ServletRequest request, ServletResponse response) {
    if (!this.isSessionIdCookieEnabled()) {
        log.debug("Session ID cookie is disabled - session id will not be acquired from a request cookie.");
        return null;
    } else if (!(request instanceof HttpServletRequest)) {
        log.debug("Current request is not an HttpServletRequest - cannot get session ID cookie.  Returning null.");
        return null;
    } else {
        HttpServletRequest httpRequest = (HttpServletRequest)request;
        return this.getSessionIdCookie().readValue(httpRequest, WebUtils.toHttp(response));
    }
}
```


return this.getSessionIdCookie().readValue(httpRequest, WebUtils.toHttp(response));


看到这一句，终于，我们通过获取cookie来读得其中保存的sessionId


这里this.getSessionIdCookie()获取的是类中定义的cookie——private Cookie sessionIdCookie，他在构造方法中创建


```java
public DefaultWebSessionManager() {
    Cookie cookie = new SimpleCookie("JSESSIONID");
    cookie.setHttpOnly(true);
    this.sessionIdCookie = cookie;
    this.sessionIdCookieEnabled = true;
    this.sessionIdUrlRewritingEnabled = false;
}
```


之后通过readValue方法，它会在请求中寻找名为JSESSIONID的cookie并返回其值


至此我们终于**获得了SessionId**


------


那我们继续回到刚才这个函数


```java
protected Session retrieveSession(SessionKey sessionKey) throws UnknownSessionException {
    Serializable sessionId = this.getSessionId(sessionKey);
    if (sessionId == null) {
        log.debug("Unable to resolve session ID from SessionKey [{}].  Returning null to indicate a session could not be found.", sessionKey);
        return null;
    } else {
        Session s = this.retrieveSessionFromDataSource(sessionId);
        if (s == null) {
            String msg = "Could not find session with ID [" + sessionId + "]";
            throw new UnknownSessionException(msg);
        } else {
            return s;
        }
    }
}
```


如果刚才cookie中获取不到sessionI则返回null，如果获取到则执行：**`Session s = retrieveSessionFromDataSource(sessionId);` 通过SessionDao根据sessionId获取到了Session**


------


如果session不为null返回最开始的那句调用：**context.setSession(session);**


```java
public void setSession(Session session) {
    this.nullSafePut(SESSION, session);
}
```


```java
protected void nullSafePut(String key, Object value) {
    if (value != null) {
        this.put(key, value);
    }

}
```


```java
public Object put(String s, Object o) {
    return this.backingMap.put(s, o);
}
```


如上文所说的**将信息存储在backingMap**中


**至此便解析完上下文中的session，对于第一次请求没有session来说在这里（过滤时）并不会创建新的session**


##### 4）this.resolvePrincipals(context)


同理将获得到的principals存储在backingMap中，过滤时如果前面没有获得到session那么这里也将得不到principals（如果是认证时调用到此处则可以获得，差别见第四点）


##### 5）this.doCreateSubject(context)


追踪该方法可以到DefaultWebSubjectFactory类中的如下方法


```java
public Subject createSubject(SubjectContext context) {
    boolean isNotBasedOnWebSubject = context.getSubject() != null && !(context.getSubject() instanceof WebSubject);
    if (context instanceof WebSubjectContext && !isNotBasedOnWebSubject) {
        WebSubjectContext wsc = (WebSubjectContext)context;
        SecurityManager securityManager = wsc.resolveSecurityManager();
        Session session = wsc.resolveSession();
        boolean sessionEnabled = wsc.isSessionCreationEnabled();
        PrincipalCollection principals = wsc.resolvePrincipals();
        boolean authenticated = wsc.resolveAuthenticated();
        String host = wsc.resolveHost();
        ServletRequest request = wsc.resolveServletRequest();
        ServletResponse response = wsc.resolveServletResponse();
        return new WebDelegatingSubject(principals, authenticated, host, session, sessionEnabled, request, response, securityManager);
    } else {
        return super.createSubject(context);
    }
}
```


我们通过subjectContext中保存的信息，执行return new WebDelegatingSubject(principals, authenticated, host, session, sessionEnabled, request, response, securityManager)，这样就能够得到当前操作的主体，知道是谁在操作，是否已经认证了。


**至此完成subject的创建**


##### 6）this.save(subject)


最终调用的是subjectDao中的save方法


```java
public Subject save(Subject subject) {
    if (this.isSessionStorageEnabled(subject)) {
        this.saveToSession(subject);
    } else {
        log.trace("Session storage of subject state for Subject [{}] has been disabled: identity and authentication state are expected to be initialized on every request or invocation.", subject);
    }

    return subject;
}
```


*这里暂时不往下追踪，等到下面第四点时会再次提到这个函数*


**至此createSubject执行完成创建**，主要步骤如下


1. 拿到subjectContext
2. 解析security，放入contex（map）中
3. 解析session，放入context（map）中
4. 解析principals，放入context（map）中
5. 通过subjectFactory创建subject
6. 通过sessionDAO保存到session中


**需要注意的是上面讲述的是这个方法的总体功能，但这是在过滤时调用的这个方法，其实大多数都没有实现，因为此时其实并没有获得到多少信息（除非是第二次请求，可以获得session），故创建的subject也没有多少信息。**


**如果没有获得session，则此时还没有用户身份信息，这个Subject还没有通过验证，只保留了三个属性：request，response，securityManager。**


**在过滤时没有session也不会创建session来保存subject信息，具体可以看下文认证时的createSubject来做对比。**


### 三、subject.execute()


```java
// 将该subject绑定到当前线程，并更新会话的上次访问时间以及分发合适的过滤器
subject.execute(new Callable() {
    public Object call() throws Exception {
        AbstractShiroFilter.this.updateSessionLastAccessTime(request, response);
        AbstractShiroFilter.this.executeChain(request, response, chain);
        return null;
    }
});
```


#### 1、execute()


追踪execute()，来到DelegatingSubject类


```java
public <V> V execute(Callable<V> callable) throws ExecutionException {
    Callable associated = this.associateWith(callable);

    try {
        return associated.call();
    } catch (Throwable var4) {
        throw new ExecutionException(var4);
    }
}
```


this.associateWith(callable)通过参数callable创建了一个SubjectCallable对象，所以我们查看SubjectCallable类中的call方法，如下：


```java
public V call() throws Exception {
    Object var1;
    try {
        this.threadState.bind();
        var1 = this.doCall(this.callable);
    } finally {
        this.threadState.restore();
    }

    return var1;
}
```


 this.threadState.bind();方法将subject和securityManager绑定到当前线程的resources（一个map），如下


```java
public void bind() {
    SecurityManager securityManager = this.securityManager;
    if (securityManager == null) {
        securityManager = ThreadContext.getSecurityManager();
    }

    this.originalResources = ThreadContext.getResources();
    ThreadContext.remove();
    ThreadContext.bind(this.subject);
    if (securityManager != null) {
        ThreadContext.bind(securityManager);
    }

}
```


 doCall(this.callable)调用回调方法


#### 2、updateSessionLastAccessTime(request, response)


更新会话上次访问时间


```java
protected void updateSessionLastAccessTime(ServletRequest request, ServletResponse response) {
    if (!this.isHttpSessions()) {
        Subject subject = SecurityUtils.getSubject();
        if (subject != null) {
            Session session = subject.getSession(false);
            if (session != null) {
                try {
                    session.touch();
                } catch (Throwable var6) {
                    log.error("session.touch() method invocation has failed.  Unable to update the corresponding session's last access time based on the incoming request.", var6);
                }
            }
        }
    }

}
```


#### 3、executeChain(request, response, chain)


将请求分发给合适的过滤器


此部分不做详解


### 四、认证时的createSubject()


基本认证流程如下：


```java
Subject subject = SecurityUtils.getSubject();
subject.login(new UsernamePasswordToken("cyh", "123"));
```


#### 1、getSubject()


追踪源码发现getSubject最终执行的是SecurityUtils类中的这么一个方法


```java
public static Subject getSubject() {
    Subject subject = ThreadContext.getSubject();
    if (subject == null) {
        subject = (new Builder()).buildSubject();
        ThreadContext.bind(subject);
    }

    return subject;
}
```


从上文的分析中我们知道了每一次访问都会重新建立subject然后绑定到当前线程，所以当前线程中获得subject不会是null。只有在没有配置shirofilter的应用中才会出现当前线程中subject==null的情况，所以这里就可以拿到前面过滤过程中创建的subject


#### 2、subject.login()


随后通过当前的subject和token调用login函数


```java
public void login(AuthenticationToken token) throws AuthenticationException {
    clearRunAsIdentitiesInternal();
    Subject subject = securityManager.login(this, token);

    PrincipalCollection principals;

    String host = null;

    if (subject instanceof DelegatingSubject) {
        DelegatingSubject delegating = (DelegatingSubject) subject;
        //we have to do this in case there are assumed identities - we don't want to lose the 'real' principals:
        principals = delegating.principals;
        host = delegating.host;
    } else {
        principals = subject.getPrincipals();
    }

    if (principals == null || principals.isEmpty()) {
        String msg = "Principals returned from securityManager.login( token ) returned a null or " +
                "empty value.  This value must be non null and populated with one or more elements.";
        throw new IllegalStateException(msg);
    }
    this.principals = principals;
    this.authenticated = true;
    if (token instanceof HostAuthenticationToken) {
        host = ((HostAuthenticationToken) token).getHost();
    }
    if (host != null) {
        this.host = host;
    }
    Session session = subject.getSession(false);
    if (session != null) {
        this.session = decorate(session);
    } else {
        this.session = null;
    }
}
```


其在最终会通过调用realm中的认证方法获得当前用户的info


```java
Subject loggedIn = createSubject(token, info, subject);
```


**随后在此处又创建了一次Subject**


那么两次创建subject有什么区别呢


```java
protected Subject createSubject(AuthenticationToken token, AuthenticationInfo info, Subject existing) {
    SubjectContext context = createSubjectContext();
    // 保存登陆状态
    context.setAuthenticated(true);
    // 保存token
    context.setAuthenticationToken(token);
    // 保存认证的返回信息
    context.setAuthenticationInfo(info);
    context.setSecurityManager(this);
    if (existing != null) {
        // 保存subject的信息到上下文
        context.setSubject(existing);
    }
    // 开始创建subject
    return createSubject(context);
}
```


可以看到此处的创建其实就是在上下文中额外增加了一些信息，之后也是通过调用上面过滤时用的那个函数createSubject(SubjectContext subjectContext)创建的subject


```java
public Subject createSubject(SubjectContext subjectContext) {
    SubjectContext context = this.copy(subjectContext);
    context = this.ensureSecurityManager(context);
    context = this.resolveSession(context);
    context = this.resolvePrincipals(context);
    Subject subject = this.doCreateSubject(context);
    this.save(subject);
    return subject;
}
```


这里着重看save函数，前面我们进入到


```java
public Subject save(Subject subject) {
    if (isSessionStorageEnabled(subject)) {
        saveToSession(subject);
    } else {
        log.trace("Session storage of subject state for Subject [{}] has been disabled: identity and " +
                "authentication state are expected to be initialized on every request or invocation.", subject);
    }

    return subject;
}
```


如果开启session，则将subject保存至session中，这里我们接着saveToSession(subject)往下追踪，来到


```java
protected void saveToSession(Subject subject) {
    mergePrincipals(subject);
    mergeAuthenticationState(subject);
}
```


先看mergePrincipals(subject)函数


```java
protected void mergePrincipals(Subject subject) {

    PrincipalCollection currentPrincipals = null;

    if (subject.isRunAs() && subject instanceof DelegatingSubject) {
        try {
            Field field = DelegatingSubject.class.getDeclaredField("principals");
            field.setAccessible(true);
            currentPrincipals = (PrincipalCollection)field.get(subject);
        } catch (Exception e) {
            throw new IllegalStateException("Unable to access DelegatingSubject principals property.", e);
        }
    }
    if (currentPrincipals == null || currentPrincipals.isEmpty()) {
        currentPrincipals = subject.getPrincipals();
    }

    Session session = subject.getSession(false);

    if (session == null) {
        if (!isEmpty(currentPrincipals)) {
            session = subject.getSession();
            session.setAttribute(DefaultSubjectContext.PRINCIPALS_SESSION_KEY, currentPrincipals);
        }
    } else {
        PrincipalCollection existingPrincipals =
                (PrincipalCollection) session.getAttribute(DefaultSubjectContext.PRINCIPALS_SESSION_KEY);

        if (isEmpty(currentPrincipals)) {
            if (!isEmpty(existingPrincipals)) {
                session.removeAttribute(DefaultSubjectContext.PRINCIPALS_SESSION_KEY);
            }
        } else {
            if (!currentPrincipals.equals(existingPrincipals)) {
                session.setAttribute(DefaultSubjectContext.PRINCIPALS_SESSION_KEY, currentPrincipals);
            }
        }
    }
}
```


其中先通过反射获取subject中的principals属性


```java
try {
       Field field = DelegatingSubject.class.getDeclaredField("principals");
       field.setAccessible(true);
       currentPrincipals = (PrincipalCollection)field.get(subject);
}
```


显然当我们在过滤时创建的subject是没有这个属性的（除非得到了session，通过session获取保存的principals），只有在登陆认证的时候之后才有这个属性


```java
if (!isEmpty(currentPrincipals)) {
       session = subject.getSession();
       session.setAttribute(DefaultSubjectContext.PRINCIPALS_SESSION_KEY, currentPrincipals);
}
```


*以下都是针对第一次请求，没有能够获取到cookie中的session的情况*


这里表示如果currentPrincipals不为空则获取session，获取不到则创建session并保存principals，过滤时的save函数不会进去这个代码段，也就没有创建session。


到这我们就清晰了，**在最开始过滤时，显然是没有principals的，虽然前面执行了resolvePrincipals(context)函数，但是此时context上下文中是没有principals的信息的，所以过滤操作时处我们并不会去创建一个session**


**而在认证时再次执行save操作的时候，此时已经调用了认证获得了对应的info，info中解析出了principals信息在调用resolvePrincipals(context)时就保存了该信息，所以save时就会因此创建一个session，随后存放principals信息在session中**


下方的mergeAuthenticationState(subject)函数其实也同理，在认证之后会将该状态保存到session中，**然后 SessionId 写到 Cookies 中**（DefaultWebSessionManager的onstart方法）


subject 创建出来之后，暂且叫内部subject，就是把认证通过的内部subject的信息和session复制给我们外界使用的subject.login(token)的subject中，这个subject暂且叫外部subject，看下session的赋值，又进行了一次装饰（decorate），这次装饰则把session(类型为StoppingAwareProxiedSession，即是内部subject和session的合体)和外部subject绑定到一起。 


### 五、用户认证后的后续请求


除了手动调用login会进入认证，其他请求过来时也会先检测当前用户是否进行了认证（即subject.authenticated是否为true），没有则先认证，有则不需要。


现在我们已经知道使用了全局的Session为我们保存了认证状态, 并且每次请求就会从这个全局Session中取出认证状态并保存到新创建的Subject中即可，不需要再执行login，也不需调用doGetAuthenticationInfo()方法


现在, 我们只需要保证前后两次请求是从同一个Session中取的认证状态即可(即两次请求在同一个会话中)，这只要通过sessionId便可以实现，即


当用户访问其他接口时，经过过滤器时会创建Subject，在其中执行resovleSession时便会调用方法获取请求携带的cookie，随后通过Cookies 中的SessionId 获取到Session，并将 Session 中的信息合并到当前Subject中，此时当前线程的 Subject.authenticated = true， Subject.principals 保存了用户信息。


随后经过过滤器验证，如指定的是 authc 过滤器，则认为是 FormAuthenticationFilter 来执行此处请求的验证。而 FormAuthenticationFilter 的判断条件是 Subject.authenticated 是否为true，因为已经合并了session中保存的subject的认证信息，故校验通过。（如果没有session，则在此处需要重新执行login操作）

