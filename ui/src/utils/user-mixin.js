export default {
    props: ['userInfo'],
    computed: {
      isFounder() {
        return this.userInfo?.role === 'founder'
      }
    },
    methods: {
      hasAccess(id) {
        return this.userInfo && this.userInfo.id && (this.userInfo.role !== 'normal' || this.userInfo.id === id)
      }
    }
}
