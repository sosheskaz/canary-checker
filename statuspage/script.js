var app = new Vue({
  el: '#app',
  store,
  created() {
    this.$store.dispatch('fetchData')
    this.$store.dispatch('resumeAutoUpdate')
  },
  computed: {
    ...Vuex.mapState(['error','servers','lastRefreshed','checks','disableReload']),
    ...Vuex.mapGetters(['serversByNames','groupedChecks'])
  },
  methods: {
    ...Vuex.mapActions(['pauseAutoUpdate', 'resumeAutoUpdate'])
  }
})

Vue.directive('popover', {
  bind: function bsPopoverCreate(el, binding) {
    let trigger = 'hover focus';
    if (binding.modifiers.focus || binding.modifiers.hover || binding.modifiers.click) {
      const t = [];
      if (binding.modifiers.focus) t.push('focus');
      if (binding.modifiers.hover) t.push('hover');
      if (binding.modifiers.click) t.push('click');
      trigger = t.join(' ');
    }
    // Time comes as UTC from server, timeago expects local time
    // We convert from UTC to Local date
    let dateTime = new Date($(el).attr("popover-title") + " UTC");
    let t = new timeago()
    let title = t.simple(date.format(dateTime, 'YYYY-MM-DD HH:mm:ss', false), 'en_US')
    let duration = $(el).attr("popover-duration")

    let content = `${binding.value} <div class="duration">Duration: ${duration / 1000}s <br/>${dateTime}</div>`

    $(el).popover({
      title: title,
      content: content,
      placement: binding.arg,
      trigger: trigger,
      html: binding.modifiers.html
    });
  },
  unbind(el, binding) {
    $(el).popover('dispose');
  },
});


// Deprecated component
Vue.component('check-row', {
  name: 'check-row',

  template: `
    <tr>
      <td scope="row" class="align-middle"> 
        <img :src="'images/' + check.type + '.svg'" height="20px" :title="check.type"></i> 
        <span class="badge badge-secondary">{{ check.name }}</span> 
        <span>{{ check.description }}</span>
      </td>
      <td v-for="(server, serverName) in serversByNames" :key="server" class="align-middle border-right">
        <section v-if="check.checkStatuses[server]">
          <button class="btn btn-secondary btn-xs" @click="triggerSingle(server, check.key)">Trigger</button>
          <div class="float-right health">{{check.health[server].latency}} {{check.health[server].uptime}}</div>
          <br />
          <div v-for="checkStatus in check.checkStatuses[server]" :key="checkStatus.time" class="check-status-container">
            <div v-if="checkStatus.status" class="check-status check-status-pass" v-popover:auto.html="checkStatus.message" v-bind:popover-duration="checkStatus.duration"  v-bind:popover-title="checkStatus.time"></div>
            <div v-else class="check-status check-status-fail" v-popover:auto.html="checkStatus.message" v-bind:popover-duration="checkStatus.duration" v-bind:popover-title="checkStatus.time"></div>
          </div>
        </section>
      </td>
    </tr>
  `,
  props: {
    check: {
      type: Object,
      required: true,
    }
  },
  computed: {
    ...Vuex.mapState(['servers']),
    ...Vuex.mapGetters(['serversByNames'])
  },
  methods: {
    triggerSingle(server, checkKey) {
      axios
        .post('/api/triggerCheck', { server, checkKey })
        .then(() => {
          this.$store.dispatch('fetchData')
        })
        .catch((err) => {
          this.$store.commit('SET_ERROR', "Trigger error: " + err.response.data)
        })
    }
  }
})

Vue.component('check-tds', {
  template: `
    <section class="check-section">
      <div class="check-section-header">
        <button class="btn btn-info btn-xs" @click="triggerSingle" title="Trigger single check on the server">
          <i class="material-icons md-12 align-middle">send</i>
        </button>
        <div class="float-right health text-right"><div>{{check.health[server].latency}}</div> {{check.health[server].uptime}}</div>
      </div>
      <div v-for="checkStatus in check.checkStatuses[this.server]" :key="checkStatus.time" class="check-status-container">
        <div v-if="checkStatus.status" class="check-status check-status-pass" v-popover:auto.html="checkStatus.message" v-bind:popover-duration="checkStatus.duration"  v-bind:popover-title="checkStatus.time"></div>
        <div v-else class="check-status check-status-fail" v-popover:auto.html="checkStatus.message" v-bind:popover-duration="checkStatus.duration" v-bind:popover-title="checkStatus.time"></div>
      </div>
    </section>
  `,
  props: {
    check: {
      type: Object,
      required: true,
    },
    server: {
      type: String,
      required: true,
    }
  },
  computed: {
    ...Vuex.mapState(['servers']),
    ...Vuex.mapGetters(['serversByNames'])
  },
  methods: {
    triggerSingle() {
      axios
        .post('/api/triggerCheck', { server: this.server, checkKey: this.check.key })
        .then(() => {
          this.$store.dispatch('fetchData')
        })
        .catch((err) => {
          this.$store.commit('SET_ERROR', "Trigger error: " + err.response.data)
        })
    }
  }
})