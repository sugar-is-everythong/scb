<script>
    let {children} = $props();
</script>

<main id="main-box">
    {@render children()}
</main>

<style>
    #main-box {
        background-color: rgb(173 197 255 / 0.3);
        height: 100vh;
        border-radius: 6px;
        overflow: hidden;
    }
</style>

