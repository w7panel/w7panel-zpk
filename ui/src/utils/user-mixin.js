export default {
    props: ['userInfo'],
    methods: {
      hasAccess(id) {
        return this.userInfo && this.userInfo.id && (this.userInfo.role !== 'normal' || this.userInfo.id === id)
      }
    }
}